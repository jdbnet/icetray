package uk.co.jdbnet.icetray

import android.app.Application
import android.content.Intent
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import uk.co.jdbnet.icetray.data.ConfigRepository
import uk.co.jdbnet.icetray.images.ImageStore
import uk.co.jdbnet.icetray.playback.PlaybackController

class IceTrayApp : Application() {
    lateinit var configRepository: ConfigRepository
        private set
    lateinit var imageStore: ImageStore
        private set

    override fun onCreate() {
        super.onCreate()
        configRepository = ConfigRepository(this)
        imageStore = ImageStore(configRepository.imagesDirectory())
    }

    companion object {
        fun get(app: Application): IceTrayApp = app as IceTrayApp
    }
}

class BootReceiver : android.content.BroadcastReceiver() {
    override fun onReceive(context: android.content.Context, intent: Intent?) {
        if (intent?.action != Intent.ACTION_BOOT_COMPLETED) return
        val pending = goAsync()
        val app = context.applicationContext as IceTrayApp
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val config = app.configRepository.load()
                if (!config.launchOnLogin || config.lastStreamId.isBlank()) return@launch
                val view = app.configRepository.getStreams().find { it.id == config.lastStreamId } ?: return@launch
                PlaybackController.play(context, view)
            } finally {
                pending.finish()
            }
        }
    }
}
