package uk.co.jdbnet.icetray.playback

import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.Build
import androidx.media3.common.AudioAttributes
import androidx.media3.common.C
import androidx.media3.common.MediaItem
import androidx.media3.common.MediaMetadata
import androidx.media3.common.Player
import androidx.media3.common.util.UnstableApi
import androidx.media3.datasource.DefaultHttpDataSource
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.exoplayer.source.DefaultMediaSourceFactory
import androidx.media3.session.DefaultMediaNotificationProvider
import androidx.media3.session.MediaSession
import androidx.media3.session.MediaSessionService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import uk.co.jdbnet.icetray.MainActivity
import uk.co.jdbnet.icetray.data.NowPlaying
import uk.co.jdbnet.icetray.data.PlaybackState
import uk.co.jdbnet.icetray.data.StreamView
import uk.co.jdbnet.icetray.metadata.MetadataFetcher

@UnstableApi
class PlaybackService : MediaSessionService() {
    private val serviceScope = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)
    private var player: ExoPlayer? = null
    private var mediaSession: MediaSession? = null
    private val metadataFetcher = MetadataFetcher()
    private var metadataJob: Job? = null
    private var reconnectJob: Job? = null
    private var currentStream: StreamView? = null
    private var currentStreamUrl: String = ""
    private var reconnectAttempt = 0

    override fun onCreate() {
        super.onCreate()
        setMediaNotificationProvider(DefaultMediaNotificationProvider(this))
        PlaybackController.attachService(this)
        val exoPlayer = buildPlayer()
        player = exoPlayer
        mediaSession = MediaSession.Builder(this, exoPlayer)
            .setSessionActivity(
                PendingIntent.getActivity(
                    this,
                    0,
                    Intent(this, MainActivity::class.java),
                    PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT,
                ),
            )
            .build()
        exoPlayer.addListener(object : Player.Listener {
            override fun onIsPlayingChanged(isPlaying: Boolean) {
                publishState()
            }

            override fun onPlaybackStateChanged(playbackState: Int) {
                publishState()
                if (playbackState == Player.STATE_ENDED || playbackState == Player.STATE_IDLE) {
                    stopMetadataPolling()
                }
                if (playbackState == Player.STATE_IDLE && currentStreamUrl.isNotBlank()) {
                    scheduleReconnect()
                }
            }

            override fun onPlayerError(error: androidx.media3.common.PlaybackException) {
                scheduleReconnect()
            }
        })
        PlaybackController.onPlayerReady(exoPlayer, mediaSession!!)
    }

    override fun onGetSession(controllerInfo: MediaSession.ControllerInfo): MediaSession? = mediaSession

    override fun onDestroy() {
        metadataJob?.cancel()
        reconnectJob?.cancel()
        mediaSession?.release()
        mediaSession = null
        player?.release()
        player = null
        serviceScope.cancel()
        PlaybackController.detachService()
        super.onDestroy()
    }

    fun playStream(stream: StreamView, volume: Int) {
        currentStream = stream
        currentStreamUrl = stream.url
        reconnectAttempt = 0
        reconnectJob?.cancel()
        val exoPlayer = player ?: return
        exoPlayer.setMediaItem(
            MediaItem.Builder()
                .setUri(stream.url)
                .setMediaId(stream.id)
                .setMediaMetadata(
                    MediaMetadata.Builder()
                        .setTitle(stream.name)
                        .setArtist(stream.name)
                        .build(),
                )
                .build(),
        )
        exoPlayer.volume = volume / 100f
        exoPlayer.prepare()
        exoPlayer.playWhenReady = true
        updateSessionMetadata(stream.name, stream.name, stream.imagePath)
        startMetadataPolling(stream.url, stream)
        publishState()
    }

    fun pause() {
        player?.pause()
        publishState()
    }

    fun resume() {
        player?.play()
        publishState()
    }

    fun stopPlayback() {
        reconnectJob?.cancel()
        stopMetadataPolling()
        currentStream = null
        currentStreamUrl = ""
        player?.stop()
        player?.clearMediaItems()
        PlaybackController.updateNowPlaying(NowPlaying())
        publishState()
        stopForeground(STOP_FOREGROUND_REMOVE)
        stopSelf()
    }

    fun setVolume(volume: Int) {
        player?.volume = volume.coerceIn(0, 100) / 100f
        publishState()
    }

    private fun buildPlayer(): ExoPlayer {
        val dataSourceFactory = DefaultHttpDataSource.Factory()
            .setUserAgent("IceTray-Android")
            .setAllowCrossProtocolRedirects(true)
        val mediaSourceFactory = DefaultMediaSourceFactory(this).setDataSourceFactory(dataSourceFactory)
        return ExoPlayer.Builder(this)
            .setMediaSourceFactory(mediaSourceFactory)
            .setAudioAttributes(
                AudioAttributes.Builder()
                    .setUsage(C.USAGE_MEDIA)
                    .setContentType(C.AUDIO_CONTENT_TYPE_MUSIC)
                    .build(),
                true,
            )
            .setHandleAudioBecomingNoisy(true)
            .build()
    }

    private fun publishState() {
        val exoPlayer = player ?: return
        val stream = currentStream
        PlaybackController.updatePlaybackState(
            PlaybackState(
                playing = exoPlayer.isPlaying,
                paused = !exoPlayer.isPlaying && exoPlayer.playbackState != Player.STATE_IDLE,
                streamId = stream?.id.orEmpty(),
                volume = (exoPlayer.volume * 100).toInt().coerceIn(0, 100),
            ),
        )
    }

    private fun startMetadataPolling(streamUrl: String, stream: StreamView) {
        metadataJob?.cancel()
        metadataJob = serviceScope.launch {
            while (isActive) {
                runCatching { metadataFetcher.fetch(streamUrl) }
                    .onSuccess { np ->
                        val merged = if (np.station.isBlank()) {
                            np.copy(station = stream.name)
                        } else {
                            np
                        }
                        updateSessionMetadata(
                            merged.title.ifBlank { stream.name },
                            merged.station.ifBlank { stream.name },
                            stream.imagePath,
                        )
                        PlaybackController.updateNowPlaying(merged)
                    }
                delay(10_000)
            }
        }
    }

    private fun stopMetadataPolling() {
        metadataJob?.cancel()
        metadataJob = null
    }

    private fun updateSessionMetadata(title: String, artist: String, imagePath: String?) {
        val builder = MediaMetadata.Builder()
            .setTitle(title)
            .setArtist(artist)
        if (!imagePath.isNullOrBlank()) {
            builder.setArtworkUri(Uri.parse("file://$imagePath"))
        }
        val metadata = builder.build()
        player?.let { exoPlayer ->
            val current = exoPlayer.currentMediaItem
            if (current != null) {
                exoPlayer.replaceMediaItem(
                    exoPlayer.currentMediaItemIndex,
                    current.buildUpon().setMediaMetadata(metadata).build(),
                )
            }
        }
    }

    private fun scheduleReconnect() {
        if (currentStreamUrl.isBlank()) return
        reconnectJob?.cancel()
        reconnectJob = serviceScope.launch {
            val delayMs = minOf(30_000L, 1_000L shl minOf(reconnectAttempt, 5))
            delay(delayMs)
            reconnectAttempt++
            val stream = currentStream ?: return@launch
            playStream(stream, (player?.volume ?: 1f).times(100).toInt())
        }
    }

    companion object {
        const val ACTION_START = "uk.co.jdbnet.icetray.action.START"
        const val EXTRA_STREAM_ID = "stream_id"
        const val EXTRA_STREAM_NAME = "stream_name"
        const val EXTRA_STREAM_URL = "stream_url"
        const val EXTRA_STREAM_IMAGE = "stream_image"
        const val EXTRA_VOLUME = "volume"
    }
}
