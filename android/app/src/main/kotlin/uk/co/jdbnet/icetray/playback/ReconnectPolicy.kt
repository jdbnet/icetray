package uk.co.jdbnet.icetray.playback

import androidx.media3.common.Player

object ReconnectPolicy {
    const val MAX_BACKOFF_MS = 30_000L
    const val STALL_TIMEOUT_MS = 15_000L

    fun backoffMs(attempt: Int): Long {
        val shift = attempt.coerceIn(0, 5)
        return minOf(MAX_BACKOFF_MS, 1_000L shl shift)
    }

    fun shouldReconnectAfterState(
        hasStream: Boolean,
        playWhenReady: Boolean,
        playbackState: Int,
    ): Boolean {
        if (!hasStream || !playWhenReady) return false
        return playbackState == Player.STATE_ENDED
    }

    fun shouldReconnectAfterError(hasStream: Boolean, playWhenReady: Boolean): Boolean {
        return hasStream && playWhenReady
    }

    fun shouldReconnectAfterStall(
        hasStream: Boolean,
        playWhenReady: Boolean,
        isPlaying: Boolean,
        playbackState: Int,
        reconnectAlreadyScheduled: Boolean,
    ): Boolean {
        if (!hasStream || !playWhenReady || isPlaying || reconnectAlreadyScheduled) return false
        return playbackState == Player.STATE_BUFFERING ||
            playbackState == Player.STATE_ENDED ||
            playbackState == Player.STATE_IDLE
    }

    fun shouldReconnectOnNetworkAvailable(
        hasStream: Boolean,
        playWhenReady: Boolean,
        isPlaying: Boolean,
    ): Boolean {
        return hasStream && playWhenReady && !isPlaying
    }

    fun shouldRetryOnResume(playbackState: Int): Boolean {
        return playbackState == Player.STATE_ENDED || playbackState == Player.STATE_IDLE
    }
}
