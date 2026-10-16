package uk.co.jdbnet.icetray

import android.content.ClipData
import android.content.Intent
import android.os.Handler
import android.os.Looper
import android.util.Log
import androidx.core.content.FileProvider
import org.json.JSONObject
import java.io.File

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
    external fun nativeExoReady()

    @JvmStatic
    fun onSessionUpdate(json: String) {
        val payload = JSONObject(json)
        Log.i(
            "IceTray",
            "session update playing=${payload.optBoolean("playing")} paused=${payload.optBoolean("paused")} title=${payload.optString("title")}",
        )
        PlaybackSessionHub.dispatch(payload)
    }

    @JvmStatic
    fun shareExportFile(path: String) {
        val context = PlaybackSessionHub.appContext() ?: return
        Handler(Looper.getMainLooper()).post {
            val file = File(path)
            val uri = FileProvider.getUriForFile(context, "${context.packageName}.fileprovider", file)
            val send = Intent(Intent.ACTION_SEND).apply {
                type = "application/zip"
                putExtra(Intent.EXTRA_STREAM, uri)
                putExtra(Intent.EXTRA_TITLE, file.name)
                clipData = ClipData.newRawUri("IceTray backup", uri)
                addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
            }
            val chooser = Intent.createChooser(send, "Export streams").apply {
                addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
            }
            context.startActivity(chooser)
        }
    }
}
