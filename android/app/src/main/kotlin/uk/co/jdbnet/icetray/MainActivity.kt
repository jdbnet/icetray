package uk.co.jdbnet.icetray

import android.content.Intent
import android.graphics.Color
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.SystemBarStyle
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.lifecycle.lifecycleScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import uk.co.jdbnet.icetray.playback.PlaybackController
import uk.co.jdbnet.icetray.ui.IceTrayTheme
import uk.co.jdbnet.icetray.ui.MainScreen

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge(
            statusBarStyle = SystemBarStyle.dark(Color.TRANSPARENT),
            navigationBarStyle = SystemBarStyle.dark(Color.TRANSPARENT),
        )
        setContent {
            IceTrayTheme {
                MainScreen()
            }
        }
        maybeResumeFromIntent(intent)
    }

    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        maybeResumeFromIntent(intent)
    }

    override fun onStart() {
        super.onStart()
        PlaybackController.bind(this)
    }

    override fun onStop() {
        PlaybackController.unbind(this)
        super.onStop()
    }

    private fun maybeResumeFromIntent(intent: Intent?) {
        if (intent == null) return
        val resume = intent.getBooleanExtra(EXTRA_RESUME_LAST_STREAM, false) ||
            intent.action == ACTION_RESUME_LAST_STREAM
        if (!resume) return
        intent.removeExtra(EXTRA_RESUME_LAST_STREAM)
        intent.action = Intent.ACTION_MAIN

        val requestedId = intent.getStringExtra(EXTRA_RESUME_STREAM_ID).orEmpty()
        getSystemService(android.app.NotificationManager::class.java)
            ?.cancel(BootReceiver.NOTIFICATION_ID)

        lifecycleScope.launch(Dispatchers.IO) {
            val app = application as IceTrayApp
            val config = app.configRepository.load()
            val id = requestedId.ifBlank { config.lastStreamId }
            if (id.isBlank()) return@launch
            val stream = app.configRepository.getStreams().find { it.id == id } ?: return@launch
            withContext(Dispatchers.Main) {
                PlaybackController.play(this@MainActivity, stream)
            }
        }
    }

    companion object {
        const val ACTION_RESUME_LAST_STREAM = "uk.co.jdbnet.icetray.action.RESUME_LAST_STREAM"
        const val EXTRA_RESUME_LAST_STREAM = "resume_last_stream"
        const val EXTRA_RESUME_STREAM_ID = "resume_stream_id"
    }
}
