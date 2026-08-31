package uk.co.jdbnet.icetray.playback

import androidx.media3.common.Player
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

class ReconnectPolicyTest {
    @Test
    fun backoffStartsAtOneSecondAndCapsAtThirty() {
        assertEquals(1_000L, ReconnectPolicy.backoffMs(0))
        assertEquals(2_000L, ReconnectPolicy.backoffMs(1))
        assertEquals(ReconnectPolicy.MAX_BACKOFF_MS, ReconnectPolicy.backoffMs(5))
        assertEquals(ReconnectPolicy.MAX_BACKOFF_MS, ReconnectPolicy.backoffMs(8))
    }

    @Test
    fun liveStreamEndTriggersReconnectWhileUserStillWantsPlayback() {
        assertTrue(
            ReconnectPolicy.shouldReconnectAfterState(
                hasStream = true,
                playWhenReady = true,
                playbackState = Player.STATE_ENDED,
            ),
        )
    }

    @Test
    fun pausedOrStoppedPlaybackDoesNotReconnectOnEnded() {
        assertFalse(
            ReconnectPolicy.shouldReconnectAfterState(
                hasStream = true,
                playWhenReady = false,
                playbackState = Player.STATE_ENDED,
            ),
        )
        assertFalse(
            ReconnectPolicy.shouldReconnectAfterState(
                hasStream = false,
                playWhenReady = true,
                playbackState = Player.STATE_ENDED,
            ),
        )
        assertFalse(
            ReconnectPolicy.shouldReconnectAfterState(
                hasStream = true,
                playWhenReady = true,
                playbackState = Player.STATE_READY,
            ),
        )
    }

    @Test
    fun stallReconnectsWhenBufferingWithoutAudio() {
        assertTrue(
            ReconnectPolicy.shouldReconnectAfterStall(
                hasStream = true,
                playWhenReady = true,
                isPlaying = false,
                playbackState = Player.STATE_BUFFERING,
                reconnectAlreadyScheduled = false,
            ),
        )
        assertFalse(
            ReconnectPolicy.shouldReconnectAfterStall(
                hasStream = true,
                playWhenReady = true,
                isPlaying = true,
                playbackState = Player.STATE_BUFFERING,
                reconnectAlreadyScheduled = false,
            ),
        )
        assertFalse(
            ReconnectPolicy.shouldReconnectAfterStall(
                hasStream = true,
                playWhenReady = true,
                isPlaying = false,
                playbackState = Player.STATE_BUFFERING,
                reconnectAlreadyScheduled = true,
            ),
        )
    }

    @Test
    fun networkRestoreRetriesWhenPlaybackHasStopped() {
        assertTrue(
            ReconnectPolicy.shouldReconnectOnNetworkAvailable(
                hasStream = true,
                playWhenReady = true,
                isPlaying = false,
            ),
        )
        assertFalse(
            ReconnectPolicy.shouldReconnectOnNetworkAvailable(
                hasStream = true,
                playWhenReady = true,
                isPlaying = true,
            ),
        )
    }

    @Test
    fun resumeRetriesEndedOrIdlePlayer() {
        assertTrue(ReconnectPolicy.shouldRetryOnResume(Player.STATE_ENDED))
        assertTrue(ReconnectPolicy.shouldRetryOnResume(Player.STATE_IDLE))
        assertFalse(ReconnectPolicy.shouldRetryOnResume(Player.STATE_READY))
    }
}
