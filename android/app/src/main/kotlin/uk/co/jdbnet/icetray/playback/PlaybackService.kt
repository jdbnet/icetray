package uk.co.jdbnet.icetray.playback

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Intent
import android.content.pm.ApplicationInfo
import android.content.pm.ServiceInfo
import android.net.ConnectivityManager
import android.net.Network
import android.net.Uri
import android.os.Build
import android.util.Log
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
import androidx.media3.session.MediaSession
import androidx.media3.session.MediaSessionService
import androidx.media3.session.SessionResult
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
    private val metadataFetcher = MetadataFetcher()
    private var metadataJob: Job? = null
    private var reconnectJob: Job? = null
    private var stallJob: Job? = null
    private var currentStream: StreamView? = null
    private var reconnectAttempt = 0
    private var retryingPlayback = false
    private var networkCallback: ConnectivityManager.NetworkCallback? = null

    override fun onCreate() {
        super.onCreate()
        ensureNotificationChannel()
        val mediaNotificationProvider = IceTrayMediaNotificationProvider(
            this,
            NOTIFICATION_CHANNEL_ID,
            R.string.playback_notification_channel,
        )
        mediaNotificationProvider.setSmallIcon(R.drawable.ic_notification)
        setMediaNotificationProvider(mediaNotificationProvider)
        PlaybackController.attachService(this)
        registerNetworkCallback()
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        super.onStartCommand(intent, flags, startId)
        when (intent?.action) {
            ACTION_STOP -> {
                stopPlayback()
                return START_NOT_STICKY
            }
            ACTION_START -> {
                val stream = streamFromIntent(intent)
                promoteToForeground(stream?.name ?: getString(R.string.app_name))
                stream?.let { playStream(it) }
                return START_STICKY
            }
            else -> return START_STICKY
        }
    }

    override fun onGetSession(controllerInfo: MediaSession.ControllerInfo): MediaSession? {
        ensurePlayerInitialized()
        return mediaSession
    }

    override fun onUpdateNotification(session: MediaSession, startInForegroundRequired: Boolean) {
        debugLog("onUpdateNotification startInForegroundRequired=$startInForegroundRequired")
        super.onUpdateNotification(session, startInForegroundRequired)
        getSystemService(NotificationManager::class.java)?.cancel(PLACEHOLDER_NOTIFICATION_ID)
    }

    override fun onDestroy() {
        metadataJob?.cancel()
        reconnectJob?.cancel()
        stallJob?.cancel()
        unregisterNetworkCallback()
        mediaSession?.release()
        mediaSession = null
        player?.release()
        player = null
        serviceScope.cancel()
        PlaybackController.detachService()
        super.onDestroy()
    }

    fun playStream(stream: StreamView) {
        ensurePlayerInitialized()
        currentStream = stream
        reconnectAttempt = 0
        reconnectJob?.cancel()
        reconnectJob = null
        retryCurrentStream(stream)
        startMetadataPolling(stream.url, stream)
        watchForStall()
        publishState()
    }

    fun pause() {
        reconnectJob?.cancel()
        reconnectJob = null
        stallJob?.cancel()
        player?.pause()
        publishState()
    }

    fun resume() {
        val exoPlayer = player
        val stream = currentStream
        if (exoPlayer != null && stream != null && ReconnectPolicy.shouldRetryOnResume(exoPlayer.playbackState)) {
            retryCurrentStream(stream)
        } else {
            exoPlayer?.play()
        }
        watchForStall()
        publishState()
    }

    fun stopPlayback() {
        reconnectJob?.cancel()
        reconnectJob = null
        stallJob?.cancel()
        stallJob = null
        stopMetadataPolling()
        currentStream = null
        reconnectAttempt = 0
        player?.let { exoPlayer ->
            exoPlayer.playWhenReady = false
            exoPlayer.stop()
            exoPlayer.clearMediaItems()
        }
        PlaybackController.updateNowPlaying(NowPlaying())
        publishState()
        stopForeground(STOP_FOREGROUND_REMOVE)
        stopSelf()
    }

    private fun ensureNotificationChannel() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val manager = getSystemService(NotificationManager::class.java) ?: return
        val existing = manager.getNotificationChannel(NOTIFICATION_CHANNEL_ID)
        if (existing != null) {
            if (existing.importance < NotificationManager.IMPORTANCE_DEFAULT) {
                manager.deleteNotificationChannel(NOTIFICATION_CHANNEL_ID)
            } else {
                return
            }
        }
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
        val notification = NotificationCompat.Builder(this, NOTIFICATION_CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(title)
            .setContentText(getString(R.string.playback_notification_channel))
            .setCategory(NotificationCompat.CATEGORY_TRANSPORT)
            .setOngoing(true)
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
        debugLog("placeholder foreground started id=$PLACEHOLDER_NOTIFICATION_ID")
    }

    private fun ensurePlayerInitialized() {
        if (player != null) return
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
            .setCallback(sessionCallback())
            .build()
        addSession(mediaSession!!)
        debugLog("media session created and added id=${mediaSession?.id}")
        exoPlayer.addListener(object : Player.Listener {
            override fun onIsPlayingChanged(isPlaying: Boolean) {
                if (isPlaying) {
                    reconnectAttempt = 0
                }
                publishState()
            }

            override fun onPlaybackStateChanged(playbackState: Int) {
                publishState()
                if (retryingPlayback) return
                val exo = player ?: return
                if (ReconnectPolicy.shouldReconnectAfterState(
                        currentStream != null,
                        exo.playWhenReady,
                        playbackState,
                    )
                ) {
                    debugLog("stream ended, scheduling reconnect")
                    scheduleReconnect()
                }
            }

            override fun onPlayerError(error: androidx.media3.common.PlaybackException) {
                Log.w(TAG, "player error ${error.errorCodeName}", error)
                val exo = player
                if (ReconnectPolicy.shouldReconnectAfterError(
                        currentStream != null,
                        exo?.playWhenReady != false,
                    )
                ) {
                    scheduleReconnect()
                }
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
                    .build()
            }

            override fun onPostConnect(session: MediaSession, controller: MediaSession.ControllerInfo) {
                session.setAvailableCommands(
                    controller,
                    MediaSession.ConnectionResult.DEFAULT_SESSION_COMMANDS,
                    availablePlayerCommands(),
                )
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
            .setConnectTimeoutMs(10_000)
            .setReadTimeoutMs(15_000)
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
            .setWakeMode(C.WAKE_MODE_NETWORK)
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

    private fun streamMetadata(title: String, artist: String, imagePath: String?): MediaMetadata {
        val builder = MediaMetadata.Builder()
            .setTitle(title)
            .setDisplayTitle(title)
            .setArtist(artist)
            .setAlbumTitle(artist)
        if (!imagePath.isNullOrBlank()) {
            builder.setArtworkUri(Uri.parse("file://$imagePath"))
        }
        return builder.build()
    }

    private fun updateSessionMetadata(title: String, artist: String, imagePath: String?) {
        // Playlist metadata updates the session without replacing the MediaItem, which would
        // tear down the live HTTP connection on Icecast streams.
        player?.playlistMetadata = streamMetadata(title, artist, imagePath)
    }

    private fun retryCurrentStream(stream: StreamView) {
        val exoPlayer = player ?: return
        retryingPlayback = true
        try {
            exoPlayer.stop()
            exoPlayer.setMediaItem(
                MediaItem.Builder()
                    .setUri(stream.url)
                    .setMediaId(stream.id)
                    .setMediaMetadata(streamMetadata(stream.name, stream.name, stream.imagePath))
                    .build(),
            )
            exoPlayer.volume = 1f
            exoPlayer.prepare()
            exoPlayer.playWhenReady = true
        } finally {
            retryingPlayback = false
        }
    }

    private fun scheduleReconnect(immediate: Boolean = false) {
        if (currentStream == null) return
        val exoPlayer = player
        if (exoPlayer != null && !exoPlayer.playWhenReady) return
        if (!immediate && reconnectJob?.isActive == true) return
        reconnectJob?.cancel()
        reconnectJob = serviceScope.launch {
            val delayMs = if (immediate) 0L else ReconnectPolicy.backoffMs(reconnectAttempt)
            if (delayMs > 0) delay(delayMs)
            if (!isActive) return@launch
            reconnectAttempt++
            withContext(Dispatchers.Main) {
                if (!isActive) return@withContext
                val current = currentStream ?: return@withContext
                val exo = player
                if (exo != null && !exo.playWhenReady) return@withContext
                debugLog("retrying stream ${current.name} attempt=$reconnectAttempt")
                retryCurrentStream(current)
            }
        }
    }

    private fun watchForStall() {
        stallJob?.cancel()
        stallJob = serviceScope.launch {
            while (isActive) {
                delay(ReconnectPolicy.STALL_TIMEOUT_MS)
                withContext(Dispatchers.Main) {
                    val exo = player ?: return@withContext
                    if (!ReconnectPolicy.shouldReconnectAfterStall(
                            currentStream != null,
                            exo.playWhenReady,
                            exo.isPlaying,
                            exo.playbackState,
                            reconnectJob?.isActive == true,
                        )
                    ) {
                        return@withContext
                    }
                    debugLog("playback stalled state=${exo.playbackState}, reconnecting")
                    scheduleReconnect()
                }
            }
        }
    }

    private fun registerNetworkCallback() {
        if (networkCallback != null) return
        val connectivity = getSystemService(ConnectivityManager::class.java) ?: return
        val callback = object : ConnectivityManager.NetworkCallback() {
            override fun onAvailable(network: Network) {
                serviceScope.launch {
                    withContext(Dispatchers.Main) {
                        val exo = player ?: return@withContext
                        if (!ReconnectPolicy.shouldReconnectOnNetworkAvailable(
                                currentStream != null,
                                exo.playWhenReady,
                                exo.isPlaying,
                            )
                        ) {
                            return@withContext
                        }
                        debugLog("network available, retrying stream")
                        scheduleReconnect(immediate = true)
                    }
                }
            }
        }
        networkCallback = callback
        connectivity.registerDefaultNetworkCallback(callback)
    }

    private fun unregisterNetworkCallback() {
        val callback = networkCallback ?: return
        networkCallback = null
        getSystemService(ConnectivityManager::class.java)?.unregisterNetworkCallback(callback)
    }

    private fun debugLog(message: String) {
        if (applicationInfo.flags and ApplicationInfo.FLAG_DEBUGGABLE != 0) {
            Log.i(TAG, message)
        }
    }

    companion object {
        private const val TAG = "IceTrayPlayback"
        private const val NOTIFICATION_CHANNEL_ID = "icetray_playback"
        private const val PLACEHOLDER_NOTIFICATION_ID = 9999

        const val ACTION_START = "uk.co.jdbnet.icetray.action.START"
        const val ACTION_STOP = "uk.co.jdbnet.icetray.action.STOP"
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

        fun intentForStop(context: android.content.Context): Intent {
            return Intent(context, PlaybackService::class.java).apply {
                action = ACTION_STOP
            }
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
