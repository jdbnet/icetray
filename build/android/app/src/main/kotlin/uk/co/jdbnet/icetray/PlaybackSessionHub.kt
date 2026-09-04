package uk.co.jdbnet.icetray

import android.content.Context
import android.content.Intent
import org.json.JSONObject
import uk.co.jdbnet.icetray.playback.PlaybackService

internal object PlaybackSessionHub {
    @Volatile
    var latest: JSONObject? = null
        private set

    fun dispatch(payload: JSONObject) {
        latest = payload
        PlaybackService.applyExternalUpdate(payload)
        val notification = payload.optBoolean("notification", false)
        val playing = payload.optBoolean("playing", false)
        val paused = payload.optBoolean("paused", false)
        if (notification && (playing || paused) && PlaybackService.instance == null) {
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
}
