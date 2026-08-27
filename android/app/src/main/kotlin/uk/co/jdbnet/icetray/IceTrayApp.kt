package uk.co.jdbnet.icetray

import android.app.Application
import android.content.pm.ApplicationInfo
import android.os.StrictMode
import uk.co.jdbnet.icetray.data.ConfigRepository
import uk.co.jdbnet.icetray.images.ImageStore

class IceTrayApp : Application() {
    lateinit var configRepository: ConfigRepository
        private set
    lateinit var imageStore: ImageStore
        private set

    override fun onCreate() {
        super.onCreate()
        if (applicationInfo.flags and ApplicationInfo.FLAG_DEBUGGABLE != 0) {
            StrictMode.setThreadPolicy(
                StrictMode.ThreadPolicy.Builder()
                    .detectAll()
                    .penaltyLog()
                    .build(),
            )
        }
        configRepository = ConfigRepository(this)
        imageStore = ImageStore(configRepository.imagesDirectory())
    }

    companion object {
        fun get(app: Application): IceTrayApp = app as IceTrayApp
    }
}
