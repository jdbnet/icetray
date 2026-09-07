package uk.co.jdbnet.icetray.cast

import android.app.Activity
import android.app.Application
import android.content.Context
import android.os.Handler
import android.os.Looper
import android.util.Log
import android.webkit.WebView
import android.widget.Toast
import androidx.mediarouter.app.MediaRouteChooserDialog
import androidx.mediarouter.app.MediaRouteControllerDialog
import com.google.android.gms.cast.MediaInfo
import com.google.android.gms.cast.MediaLoadOptions
import com.google.android.gms.cast.MediaMetadata
import com.google.android.gms.cast.MediaQueueItem
import com.google.android.gms.cast.MediaStatus
import com.google.android.gms.cast.framework.CastContext
import com.google.android.gms.cast.framework.CastSession
import com.google.android.gms.cast.framework.SessionManagerListener
import com.google.android.gms.cast.framework.media.RemoteMediaClient
import com.google.android.gms.common.ConnectionResult
import com.google.android.gms.common.GoogleApiAvailability
import com.wails.app.R
import org.json.JSONObject
import uk.co.jdbnet.icetray.NativeBridge
import java.lang.ref.WeakReference
import java.util.concurrent.Executors
import java.util.concurrent.atomic.AtomicBoolean

object CastCoordinator {
    private const val TAG = "IceTrayCast"

    private val mainHandler = Handler(Looper.getMainLooper())
    private val initExecutor = Executors.newSingleThreadExecutor()

    @Volatile
    private var appContext: Context? = null

    @Volatile
    private var webViewRef: WeakReference<WebView>? = null

    @Volatile
    private var connected: Boolean = false

    @Volatile
    private var friendlyName: String = ""

    private var lastLoadedUrl: String? = null
    private var lastTitle: String = ""
    private var lastArtist: String = ""
    private val applyingRemote = AtomicBoolean(false)
    private var sessionListener: SessionManagerListener<CastSession>? = null
    private var mediaCallback: RemoteMediaClient.Callback? = null
    private var mediaClient: RemoteMediaClient? = null

    fun isConnected(): Boolean = connected

    fun deviceName(): String = friendlyName

    fun init(app: Application) {
        appContext = app.applicationContext
        CastContext.getSharedInstance(app, initExecutor)
            .addOnFailureListener { err ->
                Log.w(TAG, "CastContext init failed", err)
            }
    }

    fun attach(webView: WebView) {
        webViewRef = WeakReference(webView)
        val context = webView.context.applicationContext
        val castContext = castContextOrNull(context) ?: return
        ensureSessionListener(castContext)
        val session = castContext.sessionManager.currentCastSession
        if (session != null && session.isConnected) {
            onCastConnected(session)
        } else {
            pushJsState()
        }
    }

    fun detach(webView: WebView) {
        val current = webViewRef?.get()
        if (current === webView) {
            webViewRef = null
        }
    }

    fun pushJsState() {
        val webView = webViewRef?.get() ?: return
        val json = JSONObject()
            .put("connected", connected)
            .put("deviceName", friendlyName)
        val script = "window.dispatchEvent(new CustomEvent('icetray-cast',{detail:$json}));"
        webView.post { webView.evaluateJavascript(script, null) }
    }

    fun showDialog(activity: Activity) {
        val availability = GoogleApiAvailability.getInstance()
            .isGooglePlayServicesAvailable(activity)
        if (availability != ConnectionResult.SUCCESS) {
            Toast.makeText(activity, R.string.cast_unavailable, Toast.LENGTH_LONG).show()
            return
        }
        val castContext = castContextOrNull(activity)
        if (castContext == null) {
            Toast.makeText(activity, R.string.cast_unavailable, Toast.LENGTH_LONG).show()
            return
        }
        val session = castContext.sessionManager.currentCastSession
        val selector = castContext.mergedSelector
        if (session != null && session.isConnected) {
            MediaRouteControllerDialog(activity).show()
        } else if (selector != null) {
            MediaRouteChooserDialog(activity).apply {
                routeSelector = selector
            }.show()
        }
    }

    fun onSessionPayload(payload: JSONObject) {
        mainHandler.post { applyPayloadOnMain(payload) }
    }

    private fun applyPayloadOnMain(payload: JSONObject) {
        val client = mediaClient ?: remoteClient() ?: return
        val streamUrl = payload.optString("streamUrl")
        val playing = payload.optBoolean("playing", false)
        val paused = payload.optBoolean("paused", false)
        val title = payload.optString("title")
        val artist = payload.optString("artist")

        if (streamUrl.isBlank()) {
            if (!playing && !paused) {
                lastLoadedUrl = null
                applyingRemote.set(true)
                client.stop()
            }
            return
        }

        if (playing && streamUrl != lastLoadedUrl) {
            load(client, streamUrl, title, artist)
            return
        }

        if (paused) {
            if (client.isPlaying) {
                applyingRemote.set(true)
                client.pause()
            }
            maybeUpdateMetadata(client, title, artist)
            return
        }

        if (playing) {
            if (client.isPaused) {
                applyingRemote.set(true)
                client.play()
            }
            maybeUpdateMetadata(client, title, artist)
            return
        }

        lastLoadedUrl = null
        applyingRemote.set(true)
        client.stop()
    }

    private fun load(client: RemoteMediaClient, url: String, title: String, artist: String) {
        val metadata = MediaMetadata(MediaMetadata.MEDIA_TYPE_MUSIC_TRACK).apply {
            putString(MediaMetadata.KEY_TITLE, title.ifBlank { "IceTray" })
            putString(MediaMetadata.KEY_ARTIST, artist.ifBlank { "IceTray" })
        }
        val info = MediaInfo.Builder(url)
            .setStreamType(MediaInfo.STREAM_TYPE_LIVE)
            .setContentType(guessContentType(url))
            .setMetadata(metadata)
            .build()
        val options = MediaLoadOptions.Builder().setAutoplay(true).build()
        lastLoadedUrl = url
        lastTitle = title
        lastArtist = artist
        applyingRemote.set(true)
        client.load(info, options).setResultCallback { result ->
            if (!result.status.isSuccess) {
                lastLoadedUrl = null
                Log.w(TAG, "Cast load failed: ${result.status}")
                val ctx = appContext ?: return@setResultCallback
                mainHandler.post {
                    Toast.makeText(ctx, R.string.cast_load_failed, Toast.LENGTH_LONG).show()
                }
            }
        }
    }

    private fun maybeUpdateMetadata(client: RemoteMediaClient, title: String, artist: String) {
        if (title == lastTitle && artist == lastArtist) {
            return
        }
        val current = client.mediaInfo ?: return
        val itemId = client.mediaStatus?.currentItemId ?: return
        if (itemId == MediaQueueItem.INVALID_ITEM_ID) {
            return
        }
        val metadata = MediaMetadata(MediaMetadata.MEDIA_TYPE_MUSIC_TRACK).apply {
            putString(MediaMetadata.KEY_TITLE, title.ifBlank { "IceTray" })
            putString(MediaMetadata.KEY_ARTIST, artist.ifBlank { "IceTray" })
        }
        val info = MediaInfo.Builder(current.contentId)
            .setStreamType(current.streamType)
            .setContentType(current.contentType)
            .setMetadata(metadata)
            .build()
        val item = MediaQueueItem.Builder(info).setItemId(itemId).build()
        lastTitle = title
        lastArtist = artist
        runCatching { client.queueUpdateItems(arrayOf(item), null) }
            .onFailure { Log.d(TAG, "Cast metadata update ignored: ${it.message}") }
    }

    private fun guessContentType(url: String): String {
        val path = url.substringBefore('?').lowercase()
        return when {
            path.endsWith(".ogg") || path.endsWith(".opus") -> "audio/ogg"
            path.endsWith(".aac") || path.endsWith(".m4a") -> "audio/aac"
            path.endsWith(".mp3") -> "audio/mpeg"
            else -> "audio/mpeg"
        }
    }

    private fun ensureSessionListener(castContext: CastContext) {
        if (sessionListener != null) {
            return
        }
        val listener = object : SessionManagerListener<CastSession> {
            override fun onSessionStarting(session: CastSession) = Unit

            override fun onSessionStarted(session: CastSession, sessionId: String) {
                onCastConnected(session)
            }

            override fun onSessionStartFailed(session: CastSession, error: Int) {
                Log.w(TAG, "Cast session start failed: $error")
            }

            override fun onSessionEnding(session: CastSession) = Unit

            override fun onSessionEnded(session: CastSession, error: Int) {
                onCastDisconnected()
            }

            override fun onSessionResuming(session: CastSession, sessionId: String) = Unit

            override fun onSessionResumed(session: CastSession, wasSuspended: Boolean) {
                onCastConnected(session)
            }

            override fun onSessionResumeFailed(session: CastSession, error: Int) {
                onCastDisconnected()
            }

            override fun onSessionSuspended(session: CastSession, reason: Int) = Unit
        }
        sessionListener = listener
        castContext.sessionManager.addSessionManagerListener(listener, CastSession::class.java)
    }

    private fun onCastConnected(session: CastSession) {
        connected = true
        friendlyName = session.castDevice?.friendlyName.orEmpty()
        bindMediaCallback(session)
        NativeBridge.nativeSetCasting(true)
        pushJsState()
    }

    private fun onCastDisconnected() {
        unbindMediaCallback()
        lastLoadedUrl = null
        lastTitle = ""
        lastArtist = ""
        connected = false
        friendlyName = ""
        NativeBridge.nativeSetCasting(false)
        pushJsState()
    }

    private fun bindMediaCallback(session: CastSession) {
        unbindMediaCallback()
        val client = session.remoteMediaClient ?: return
        mediaClient = client
        val callback = object : RemoteMediaClient.Callback() {
            override fun onStatusUpdated() {
                handleRemoteStatus(client)
            }
        }
        mediaCallback = callback
        client.registerCallback(callback)
    }

    private fun unbindMediaCallback() {
        val client = mediaClient
        val callback = mediaCallback
        if (client != null && callback != null) {
            client.unregisterCallback(callback)
        }
        mediaCallback = null
        mediaClient = null
    }

    private fun handleRemoteStatus(client: RemoteMediaClient) {
        if (applyingRemote.getAndSet(false)) {
            return
        }
        when (client.playerState) {
            MediaStatus.PLAYER_STATE_PAUSED -> NativeBridge.nativePause()
            MediaStatus.PLAYER_STATE_PLAYING -> NativeBridge.nativeResume()
            MediaStatus.PLAYER_STATE_IDLE -> {
                val reason = client.idleReason
                if (reason == MediaStatus.IDLE_REASON_ERROR) {
                    Log.w(TAG, "Cast player idle with error")
                    val ctx = appContext ?: return
                    Toast.makeText(ctx, R.string.cast_load_failed, Toast.LENGTH_LONG).show()
                }
            }
        }
    }

    private fun remoteClient(): RemoteMediaClient? {
        val context = appContext ?: return null
        val session = castContextOrNull(context)?.sessionManager?.currentCastSession
        return session?.remoteMediaClient
    }

    private fun castContextOrNull(context: Context): CastContext? {
        return try {
            CastContext.getSharedInstance(context)
        } catch (err: Exception) {
            Log.w(TAG, "CastContext unavailable", err)
            null
        }
    }
}
