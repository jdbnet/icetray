package uk.co.jdbnet.icetray

import android.util.Log
import org.json.JSONObject

object NativeBridge {
    @JvmStatic
    external fun nativeRegister()

    @JvmStatic
    external fun nativePause()

    @JvmStatic
    external fun nativeResume()

    @JvmStatic
    external fun nativeStop()

    @JvmStatic
    external fun nativePlayLast()

    @JvmStatic
    external fun nativePlay(streamId: String)

    @JvmStatic
    fun onSessionUpdate(json: String) {
        val payload = JSONObject(json)
        Log.i(
            "IceTray",
            "session update playing=${payload.optBoolean("playing")} paused=${payload.optBoolean("paused")} title=${payload.optString("title")}",
        )
        PlaybackSessionHub.dispatch(payload)
    }
}
