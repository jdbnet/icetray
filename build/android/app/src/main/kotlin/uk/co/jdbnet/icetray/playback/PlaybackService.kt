package uk.co.jdbnet.icetray.playback

import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.content.pm.ServiceInfo
import android.media.AudioAttributes
import android.media.AudioFocusRequest
import android.media.AudioManager
import android.net.wifi.WifiManager
import android.os.Build
import android.os.Handler
import android.os.Looper
import android.os.PowerManager
import androidx.core.app.NotificationCompat
import androidx.core.app.ServiceCompat
import androidx.core.content.ContextCompat
import androidx.media3.common.util.UnstableApi
import androidx.media3.session.DefaultMediaNotificationProvider
import androidx.media3.session.MediaSession
import androidx.media3.session.MediaSessionService
import androidx.media3.session.MediaStyleNotificationHelper
import com.wails.app.MainActivity
import com.wails.app.R
import org.json.JSONObject
import uk.co.jdbnet.icetray.PlaybackSessionHub

@UnstableApi
class PlaybackService : MediaSessionService() {
    private var player: GoBackingPlayer? = null
    private var mediaSession: MediaSession? = null
    private var audioFocusRequest: AudioFocusRequest? = null
    private var pausedForTransientFocus = false
    private var hasAudioFocus = false
    private var foregroundStarted = false
    private var wakeLock: PowerManager.WakeLock? = null
    private var wifiLock: WifiManager.WifiLock? = null
    private var noisyRegistered = false

    private val noisyReceiver = object : BroadcastReceiver() {
        override fun onReceive(context: Context?, intent: Intent?) {
            if (intent?.action == AudioManager.ACTION_AUDIO_BECOMING_NOISY) {
                uk.co.jdbnet.icetray.NativeBridge.nativePause()
            }
        }
    }

    override fun onCreate() {
        super.onCreate()
        appContext = applicationContext
        ensureNotificationChannel()
        val backing = GoBackingPlayer(Looper.getMainLooper())
        player = backing
        mediaSession = MediaSession.Builder(this, backing)
            .setSessionActivity(
                PendingIntent.getActivity(
                    this,
                    0,
                    Intent(this, MainActivity::class.java),
                    PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT,
                ),
            )
            .build()
        addSession(mediaSession!!)
        val provider = IceTrayMediaNotificationProvider(
            this,
            NOTIFICATION_CHANNEL_ID,
            R.string.playback_notification_channel,
        )
        provider.setSmallIcon(R.drawable.ic_notification)
        setMediaNotificationProvider(provider)
        instance = this
        PlaybackSessionHub.latest?.let { applyPayload(it) }
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        super.onStartCommand(intent, flags, startId)
        when (intent?.action) {
            ACTION_STOP -> {
                uk.co.jdbnet.icetray.NativeBridge.nativeStop()
                stopSelf()
                return START_NOT_STICKY
            }
            else -> {
                if (!foregroundStarted) {
                    promoteToForeground()
                }
                acquireLocks()
                requestAudioFocus()
                registerNoisy()
                return START_STICKY
            }
        }
    }

    override fun onUpdateNotification(session: MediaSession, startInForegroundRequired: Boolean) {
        super.onUpdateNotification(session, startInForegroundRequired)
        if (startInForegroundRequired) {
            foregroundStarted = true
        }
    }

    override fun onGetSession(controllerInfo: MediaSession.ControllerInfo): MediaSession? {
        return mediaSession
    }

    override fun onDestroy() {
        instance = null
        unregisterNoisy()
        abandonAudioFocus()
        releaseLocks()
        foregroundStarted = false
        mediaSession?.release()
        mediaSession = null
        player = null
        super.onDestroy()
    }

    private fun applyPayload(payload: JSONObject) {
        val playing = payload.optBoolean("playing", false)
        val paused = payload.optBoolean("paused", false)
        val apply = {
            player?.applySession(
                playing = playing,
                paused = paused,
                title = payload.optString("title"),
                artist = payload.optString("artist"),
                artworkPath = payload.optString("artworkPath").takeIf { it.isNotBlank() },
                streamId = payload.optString("streamId"),
            )
            if (!playing && !paused) {
                stopForeground(STOP_FOREGROUND_REMOVE)
                stopSelf()
            }
        }
        if (Looper.myLooper() == Looper.getMainLooper()) {
            apply()
        } else {
            Handler(Looper.getMainLooper()).post(apply)
        }
    }

    private fun promoteToForeground() {
        val session = mediaSession
        val builder = NotificationCompat.Builder(this, NOTIFICATION_CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(getString(R.string.app_name))
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
        if (session != null) {
            builder.setStyle(MediaStyleNotificationHelper.MediaStyle(session))
        }
        val notification = builder.build()
        ServiceCompat.startForeground(
            this,
            DefaultMediaNotificationProvider.DEFAULT_NOTIFICATION_ID,
            notification,
            ServiceInfo.FOREGROUND_SERVICE_TYPE_MEDIA_PLAYBACK,
        )
        foregroundStarted = true
    }

    private fun ensureNotificationChannel() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val manager = getSystemService(NotificationManager::class.java) ?: return
        val existing = manager.getNotificationChannel(NOTIFICATION_CHANNEL_ID)
        if (existing != null) return
        manager.createNotificationChannel(
            NotificationChannel(
                NOTIFICATION_CHANNEL_ID,
                getString(R.string.playback_notification_channel),
                NotificationManager.IMPORTANCE_LOW,
            ).apply {
                description = getString(R.string.playback_notification_channel)
                setShowBadge(false)
                setSound(null, null)
                enableVibration(false)
                enableLights(false)
            },
        )
    }

    private fun requestAudioFocus() {
        if (hasAudioFocus) return
        val audioManager = getSystemService(AudioManager::class.java) ?: return
        val attrs = AudioAttributes.Builder()
            .setUsage(AudioAttributes.USAGE_MEDIA)
            .setContentType(AudioAttributes.CONTENT_TYPE_MUSIC)
            .build()
        val request = AudioFocusRequest.Builder(AudioManager.AUDIOFOCUS_GAIN)
            .setAudioAttributes(attrs)
            .setAcceptsDelayedFocusGain(false)
            .setWillPauseWhenDucked(false)
            .setOnAudioFocusChangeListener { focus ->
                when (focus) {
                    AudioManager.AUDIOFOCUS_LOSS -> {
                        hasAudioFocus = false
                        pausedForTransientFocus = false
                        uk.co.jdbnet.icetray.NativeBridge.nativePause()
                    }
                    AudioManager.AUDIOFOCUS_LOSS_TRANSIENT -> {
                        pausedForTransientFocus = true
                        uk.co.jdbnet.icetray.NativeBridge.nativePause()
                    }
                    AudioManager.AUDIOFOCUS_GAIN -> {
                        hasAudioFocus = true
                        if (pausedForTransientFocus) {
                            pausedForTransientFocus = false
                            uk.co.jdbnet.icetray.NativeBridge.nativeResume()
                        }
                    }
                }
            }
            .build()
        audioFocusRequest = request
        hasAudioFocus = audioManager.requestAudioFocus(request) == AudioManager.AUDIOFOCUS_REQUEST_GRANTED
    }

    private fun abandonAudioFocus() {
        val request = audioFocusRequest ?: return
        getSystemService(AudioManager::class.java)?.abandonAudioFocusRequest(request)
        audioFocusRequest = null
        hasAudioFocus = false
        pausedForTransientFocus = false
    }

    private fun acquireLocks() {
        if (wakeLock == null) {
            wakeLock = getSystemService(PowerManager::class.java)
                ?.newWakeLock(PowerManager.PARTIAL_WAKE_LOCK, "icetray:playback")
                ?.apply { setReferenceCounted(false) }
        }
        if (wakeLock?.isHeld != true) {
            wakeLock?.acquire()
        }
        if (wifiLock == null) {
            @Suppress("DEPRECATION")
            wifiLock = applicationContext.getSystemService(WifiManager::class.java)
                ?.createWifiLock(WifiManager.WIFI_MODE_FULL_HIGH_PERF, "icetray:wifi")
                ?.apply { setReferenceCounted(false) }
        }
        if (wifiLock?.isHeld != true) {
            @Suppress("DEPRECATION")
            wifiLock?.acquire()
        }
    }

    private fun releaseLocks() {
        if (wakeLock?.isHeld == true) wakeLock?.release()
        if (wifiLock?.isHeld == true) {
            @Suppress("DEPRECATION")
            wifiLock?.release()
        }
    }

    private fun registerNoisy() {
        if (noisyRegistered) return
        ContextCompat.registerReceiver(
            this,
            noisyReceiver,
            IntentFilter(AudioManager.ACTION_AUDIO_BECOMING_NOISY),
            ContextCompat.RECEIVER_NOT_EXPORTED,
        )
        noisyRegistered = true
    }

    private fun unregisterNoisy() {
        if (!noisyRegistered) return
        runCatching { unregisterReceiver(noisyReceiver) }
        noisyRegistered = false
    }

    companion object {
        const val ACTION_SYNC = "uk.co.jdbnet.icetray.action.SYNC"
        const val ACTION_START = "uk.co.jdbnet.icetray.action.START"
        const val ACTION_STOP = "uk.co.jdbnet.icetray.action.STOP"
        private const val NOTIFICATION_CHANNEL_ID = "icetray_playback"

        @Volatile
        var appContext: Context? = null

        @Volatile
        internal var instance: PlaybackService? = null

        fun applyExternalUpdate(payload: JSONObject) {
            instance?.applyPayload(payload)
        }
    }
}
