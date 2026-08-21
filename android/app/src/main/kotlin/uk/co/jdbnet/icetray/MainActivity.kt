package uk.co.jdbnet.icetray

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import uk.co.jdbnet.icetray.ui.IceTrayTheme
import uk.co.jdbnet.icetray.ui.MainScreen

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            IceTrayTheme {
                MainScreen()
            }
        }
    }
}
