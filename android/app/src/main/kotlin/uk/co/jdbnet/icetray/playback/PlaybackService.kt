package uk.co.jdbnet.icetray.playback

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.content.pm.ServiceInfo
import android.net.Uri
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.ServiceCompat
import androidx.media3.common.AudioAttributes
import androidx.media3.common.C
import androidx.media3.common.MediaItem
import androidx.media3.common.MediaMetadata
import androidx.media3.common.Player
import androidx.media3.common.util.UnstableApi
import androidx.media3.datasource.DefaultHttpDataSource
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.exoplayer.source.DefaultMediaSourceFactory
import androidx.media3.session.CommandButton
import androidx.media3.session.DefaultMediaNotificationProvider
import androidx.media3.session.MediaSession
import androidx.media3.session.MediaSessionService
import androidx.media3.session.SessionResult
import com.google.common.collect.ImmutableList
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import uk.co.jdbnet.icetray.MainActivity
import uk.co.jdbnet.icetray.R
import uk.co.jdbnet.icetray.data.NowPlaying
import uk.co.jdbnet.icetray.data.PlaybackState
import uk.co.jdbnet.icetray.data.StreamView
import uk.co.jdbnet.icetray.metadata.MetadataFetcher

@UnstableApi
class PlaybackService : MediaSessionService() {
    private val serviceScope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    private var player: ExoPlayer? = null
    private var mediaSession: MediaSession? = null
    private var playerInitialized = false
    private val metadataFetcher = MetadataFetcher()
    private var metadataJob: Job? = null
    private var reconnectJob: Job? = null
    private var currentStream: StreamView? = null
    private var currentStreamUrl: String = ""
    private var reconnectAttempt = 0
    private var placeholderActive = false

    override fun onCreate() {
        super.onCreate()
        ensureNotificationChannel()
        val mediaNotificationProvider = DefaultMediaNotificationProvider.Builder(this)
            .setChannelId(NOTIFICATION_CHANNEL_ID)
            .setChannelName(R.string.playback_notification_channel)
            .build()
        mediaNotificationProvider.setSmallIcon(R.drawable.ic_notification)
        setMediaNotificationProvider(mediaNotificationProvider)
        PlaybackController.attachService(this)
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        super.onStartCommand(intent, flags, startId)
        val stream = intent?.takeIf { it.action == ACTION_START }?.let { streamFromIntent(it) }
        promoteToForeground(stream?.name ?: getString(R.string.app_name))
        stream?.let { playStream(it) }
        return START_STICKY
    }

    override fun onGetSession(controllerInfo: MediaSession.ControllerInfo): MediaSession? {
        ensurePlayerInitialized()
        return mediaSession
    }

    override fun onUpdateNotification(session: MediaSession, startInForegroundRequired: Boolean) {
        super.onUpdateNotification(session, startInForegroundRequired)
        dismissPlaceholderNotification()
    }

    override fun onTaskRemoved(rootIntent: Intent?) {
        if (player?.isPlaying == true) {
            return
        }
        super.onTaskRemoved(rootIntent)
    }

    override fun onDestroy() {
        metadataJob?.cancel()
        reconnectJob?.cancel()
        mediaSession?.release()
        mediaSession = null
        player?.release()
        player = null
        playerInitialized = false
        serviceScope.cancel()
        PlaybackController.detachService()
        super.onDestroy()
    }

    fun playStream(stream: StreamView) {
        ensurePlayerInitialized()
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
        exoPlayer.volume = 1f
        exoPlayer.prepare()
        exoPlayer.playWhenReady = true
        updateSessionMetadata(stream.name, stream.name, stream.imagePath)
        mediaSession?.setMediaButtonPreferences(mediaButtonPreferences())
        startMetadataPolling(stream.url, stream)
        publishState()
        mediaSession?.let { onUpdateNotification(it, /* startInForegroundRequired= */ true) }
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
        dismissPlaceholderNotification()
        stopForeground(STOP_FOREGROUND_REMOVE)
        stopSelf()
    }

    private fun ensureNotificationChannel() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val manager = getSystemService(NotificationManager::class.java) ?: return
        if (manager.getNotificationChannel(NOTIFICATION_CHANNEL_ID) != null) return
        val channel = NotificationChannel(
            NOTIFICATION_CHANNEL_ID,
            getString(R.string.playback_notification_channel),
            NotificationManager.IMPORTANCE_DEFAULT,
        ).apply {
            description = getString(R.string.playback_notification_channel)
            setShowBadge(false)
        }
        manager.createNotificationChannel(channel)
    }

    private fun promoteToForeground(title: String) {
        if (placeholderActive) return
        val notification = NotificationCompat.Builder(this, NOTIFICATION_CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(title)
            .setContentText(getString(R.string.playback_notification_channel))
            .setOngoing(true)
            .setSilent(true)
            .setContentIntent(
                PendingIntent.getActivity(
                    this,
                    0,
                    Intent(this, MainActivity::class.java),
                    PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT,
                ),
            )
            .build()
        ServiceCompat.startForeground(
            this,
            PLACEHOLDER_NOTIFICATION_ID,
            notification,
            ServiceInfo.FOREGROUND_SERVICE_TYPE_MEDIA_PLAYBACK,
        )
        placeholderActive = true
    }

    private fun dismissPlaceholderNotification() {
        if (!placeholderActive) return
        placeholderActive = false
        getSystemService(NotificationManager::class.java)?.cancel(PLACEHOLDER_NOTIFICATION_ID)
    }

    private fun ensurePlayerInitialized() {
        if (playerInitialized) return
        playerInitialized = true
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
            .setMediaButtonPreferences(mediaButtonPreferences())
            .setCallback(sessionCallback())
            .build()
        exoPlayer.addListener(object : Player.Listener {
            override fun onIsPlayingChanged(isPlaying: Boolean) {
                publishState()
            }

            override fun onPlaybackStateChanged(playbackState: Int) {
                publishState()
            }

            override fun onPlayerError(error: androidx.media3.common.PlaybackException) {
                scheduleReconnect()
            }
        })
        PlaybackController.onPlayerReady(exoPlayer, mediaSession!!)
    }

    private fun sessionCallback(): MediaSession.Callback {
        return object : MediaSession.Callback {
            override fun onConnect(
                session: MediaSession,
                controller: MediaSession.ControllerInfo,
            ): MediaSession.ConnectionResult {
                return MediaSession.ConnectionResult.AcceptedResultBuilder(session)
                    .setAvailablePlayerCommands(availablePlayerCommands())
                    .setMediaButtonPreferences(mediaButtonPreferences())
                    .build()
            }

            override fun onPostConnect(session: MediaSession, controller: MediaSession.ControllerInfo) {
                session.setAvailableCommands(
                    controller,
                    MediaSession.ConnectionResult.DEFAULT_SESSION_COMMANDS,
                    availablePlayerCommands(),
                )
                session.setMediaButtonPreferences(controller, mediaButtonPreferences())
            }

            override fun onPlayerCommandRequest(
                session: MediaSession,
                controller: MediaSession.ControllerInfo,
                playerCommand: Int,
            ): Int {
                if (playerCommand == Player.COMMAND_STOP) {
                    stopPlayback()
                    return SessionResult.RESULT_SUCCESS
                }
                return super.onPlayerCommandRequest(session, controller, playerCommand)
            }
        }
    }

    private fun availablePlayerCommands(): Player.Commands {
        return MediaSession.ConnectionResult.DEFAULT_PLAYER_COMMANDS.buildUpon()
            .add(Player.COMMAND_STOP)
            .build()
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
            .setWakeMode(C.WAKE_MODE_LOCAL)
            .build()
    }

    private fun mediaButtonPreferences(): ImmutableList<CommandButton> {
        return ImmutableList.of(
            CommandButton.Builder()
                .setPlayerCommand(Player.COMMAND_PLAY_PAUSE)
                .setDisplayName(getString(R.string.play_pause))
                .build(),
            CommandButton.Builder()
                .setPlayerCommand(Player.COMMAND_STOP)
                .setDisplayName(getString(R.string.stop))
                .setIconResId(R.drawable.ic_media_stop)
                .build(),
        )
    }

    private fun publishState() {
        val exoPlayer = player ?: return
        val stream = currentStream
        PlaybackController.updatePlaybackState(
            PlaybackState(
                playing = exoPlayer.isPlaying,
                paused = !exoPlayer.isPlaying && exoPlayer.playbackState != Player.STATE_IDLE,
                streamId = stream?.id.orEmpty(),
                volume = 100,
            ),
        )
    }

    private fun startMetadataPolling(streamUrl: String, stream: StreamView) {
        metadataJob?.cancel()
        metadataJob = serviceScope.launch {
            while (isActive) {
                val merged = runCatching { metadataFetcher.fetch(streamUrl) }
                    .getOrNull()
                    ?.let { np ->
                        if (np.station.isBlank()) np.copy(station = stream.name) else np
                    }
                if (merged != null) {
                    withContext(Dispatchers.Main) {
                        updateSessionMetadata(
                            merged.title.ifBlank { stream.name },
                            merged.station.ifBlank { stream.name },
                            stream.imagePath,
                        )
                        PlaybackController.updateNowPlaying(merged)
                    }
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
            val current = exoPlayer.currentMediaItem ?: return
            exoPlayer.replaceMediaItem(
                exoPlayer.currentMediaItemIndex,
                current.buildUpon().setMediaMetadata(metadata).build(),
            )
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
            withContext(Dispatchers.Main) {
                playStream(stream)
            }
        }
    }

    companion object {
        private const val NOTIFICATION_CHANNEL_ID = "icetray_playback"
        private const val PLACEHOLDER_NOTIFICATION_ID = 9999

        const val ACTION_START = "uk.co.jdbnet.icetray.action.START"
        const val EXTRA_STREAM_ID = "stream_id"
        const val EXTRA_STREAM_NAME = "stream_name"
        const val EXTRA_STREAM_URL = "stream_url"
        const val EXTRA_STREAM_IMAGE = "stream_image"

        fun streamFromIntent(intent: Intent): StreamView? {
            val id = intent.getStringExtra(EXTRA_STREAM_ID).orEmpty()
            val name = intent.getStringExtra(EXTRA_STREAM_NAME).orEmpty()
            val url = intent.getStringExtra(EXTRA_STREAM_URL).orEmpty()
            if (id.isBlank() || name.isBlank() || url.isBlank()) return null
            val imagePath = intent.getStringExtra(EXTRA_STREAM_IMAGE)?.takeIf { it.isNotBlank() }
            return StreamView(
                id = id,
                name = name,
                url = url,
                imagePath = imagePath,
            )
        }

        fun intentForStream(context: android.content.Context, stream: StreamView): Intent {
            return Intent(context, PlaybackService::class.java).apply {
                action = ACTION_START
                putExtra(EXTRA_STREAM_ID, stream.id)
                putExtra(EXTRA_STREAM_NAME, stream.name)
                putExtra(EXTRA_STREAM_URL, stream.url)
                if (!stream.imagePath.isNullOrBlank()) {
                    putExtra(EXTRA_STREAM_IMAGE, stream.imagePath)
                }
            }
        }
    }
}
