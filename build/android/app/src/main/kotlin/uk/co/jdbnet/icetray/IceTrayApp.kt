package uk.co.jdbnet.icetray

import android.app.Application
import uk.co.jdbnet.icetray.cast.CastCoordinator

class IceTrayApp : Application() {
    override fun onCreate() {
        super.onCreate()
        CastCoordinator.init(this)
    }
}
