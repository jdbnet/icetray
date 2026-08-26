package uk.co.jdbnet.icetray.playback

import androidx.media3.common.Player
import androidx.media3.common.util.UnstableApi
import androidx.media3.session.CommandButton
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotEquals
import org.junit.Test

@UnstableApi
class IceTrayMediaNotificationProviderTest {
    @Test
    fun pauseAndStopButtonsHaveIconResources() {
        val specs = iceTrayMediaButtonSpecs(
            playLabel = "Play",
            pauseLabel = "Pause",
            stopLabel = "Stop",
            includePlayPause = true,
            includeStop = true,
            showPauseButton = true,
        )

        assertEquals(2, specs.size)
        specs.forEach { spec ->
            assertNotEquals(0, spec.customIconResId)
            assertNotEquals(CommandButton.ICON_UNDEFINED, spec.icon)
        }
        assertEquals(CommandButton.ICON_PAUSE, specs[0].icon)
        assertEquals(Player.COMMAND_PLAY_PAUSE, specs[0].playerCommand)
        assertEquals("Pause", specs[0].displayName)
        assertEquals(CommandButton.ICON_STOP, specs[1].icon)
        assertEquals(Player.COMMAND_STOP, specs[1].playerCommand)
        assertEquals("Stop", specs[1].displayName)
    }

    @Test
    fun playButtonHasIconResourceWhenPaused() {
        val specs = iceTrayMediaButtonSpecs(
            playLabel = "Play",
            pauseLabel = "Pause",
            stopLabel = "Stop",
            includePlayPause = true,
            includeStop = false,
            showPauseButton = false,
        )

        assertEquals(1, specs.size)
        assertEquals(CommandButton.ICON_PLAY, specs[0].icon)
        assertNotEquals(0, specs[0].customIconResId)
        assertEquals("Play", specs[0].displayName)
    }
}
