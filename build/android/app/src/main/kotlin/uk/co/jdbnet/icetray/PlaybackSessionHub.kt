package uk.co.jdbnet.icetray

import android.content.Context
import android.content.Intent
import org.json.JSONObject
import uk.co.jdbnet.icetray.cast.CastCoordinator
import uk.co.jdbnet.icetray.playback.PlaybackService

internal object PlaybackSessionHub {
    @Volatile
    var latest: JSONObject? = null
        private set

    fun dispatch(payload: JSONObject) {
        latest = payload
        if (payload.optBoolean("casting", false)) {
            CastCoordinator.onSessionPayload(payload)
            PlaybackService.dismissForCast()
            return
        }
        PlaybackService.applyExternalUpdate(payload)
        val notification = payload.optBoolean("notification", false)
        val playing = payload.optBoolean("playing", false)
        val paused = payload.optBoolean("paused", false)
        val loading = payload.optBoolean("loading", false)
        if (notification && (playing || paused || loading) && PlaybackService.instance == null) {
            val context = PlaybackService.appContext ?: return
            val intent = Intent(context, PlaybackService::class.java).apply {
                action = PlaybackService.ACTION_START
            }
            context.startForegroundService(intent)
        }
    }

    fun bindContext(context: Context) {
        PlaybackService.appContext = context.applicationContext
    }

    /**
     * Media3 1.11 keeps an active MediaSession while IceTray is playing locally. That session
     * can prevent Cast route discovery from presenting devices. Drop the foreground session while
     * the Cast picker is visible; local audio still comes from the Go/oto pipeline.
     */
    fun releaseLocalMediaSessionForCastPicker() {
        PlaybackService.dismissForCast()
    }

    /** Restore notification media controls if the user closed the picker without connecting. */
    fun restoreLocalMediaSessionAfterCastPicker() {
        val payload = latest ?: return
        if (payload.optBoolean("casting", false)) {
            return
        }
        dispatch(payload)
    }
}
