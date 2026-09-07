package uk.co.jdbnet.icetray.cast

import android.app.Activity
import android.webkit.JavascriptInterface

class CastJsBridge(private val activity: Activity) {
    @JavascriptInterface
    fun showDialog() {
        activity.runOnUiThread { CastCoordinator.showDialog(activity) }
    }

    @JavascriptInterface
    fun isConnected(): Boolean = CastCoordinator.isConnected()

    @JavascriptInterface
    fun deviceName(): String = CastCoordinator.deviceName()
}
