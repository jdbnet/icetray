package uk.co.jdbnet.icetray.playback

import android.content.Context
import android.net.Uri
import android.os.Handler
import android.os.Looper
import android.os.SystemClock
import android.util.Log
import androidx.media3.common.AudioAttributes
import androidx.media3.common.C
import androidx.media3.common.MediaItem
import androidx.media3.common.PlaybackException
import androidx.media3.common.Player
import androidx.media3.common.util.UnstableApi
import androidx.media3.datasource.DefaultHttpDataSource
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.exoplayer.source.DefaultMediaSourceFactory
import uk.co.jdbnet.icetray.NativeBridge
import kotlin.math.cos
import kotlin.math.sin

/**
 * Plays Icecast URLs with Media3 ExoPlayer. Station changes overlap two players and
 * ramp their volumes for the configured crossfade.
 */
@UnstableApi
class ExoPlayback(context: Context) {
    private val appContext = context.applicationContext
    private val mainHandler = Handler(Looper.getMainLooper())
    private var current: ExoPlayer? = null
    private var outgoing: ExoPlayer? = null
    private var currentUrl: String? = null
    private var fadeGeneration = 0
    private var fadeRunnable: Runnable? = null

    fun apply(playing: Boolean, paused: Boolean, loading: Boolean, url: String, crossfadeMs: Int) {
        // The Go side reports loading before ExoPlayer has reached READY, so playing is still false.
        val active = playing || loading
        if (url.isBlank() || (!active && !paused)) {
            releaseAll()
            return
        }
        if (paused) {
            pauseAll()
            return
        }
        if (currentUrl == url && current != null) {
            current?.play()
            if (current?.playbackState == Player.STATE_READY) {
                notifyReady()
            }
            return
        }
        val next = newPlayer(url)
        val previous = current
        if (previous == null) {
            current = next
            currentUrl = url
            next.playWhenReady = true
            return
        }
        if (crossfadeMs <= 0) {
            cancelFade()
            releasePlayer(outgoing)
            outgoing = previous
            current = next
            currentUrl = url
            next.volume = 1f
            next.playWhenReady = true
            var cutDone = false
            next.addListener(object : Player.Listener {
                override fun onPlaybackStateChanged(playbackState: Int) {
                    if (cutDone || playbackState != Player.STATE_READY || current !== next) {
                        return
                    }
                    cutDone = true
                    if (outgoing === previous) {
                        outgoing = null
                    }
                    releasePlayer(previous)
                    notifyReady()
                }
            })
            return
        }
        cancelFade()
        releasePlayer(outgoing)
        outgoing = previous
        current = next
        currentUrl = url
        previous.volume = 1f
        next.volume = 0f
        next.playWhenReady = true
        var fadeStarted = false
        next.addListener(object : Player.Listener {
            override fun onPlaybackStateChanged(playbackState: Int) {
                if (playbackState != Player.STATE_READY || current !== next || fadeStarted) {
                    return
                }
                fadeStarted = true
                startFade(previous, next, crossfadeMs)
            }
        })
    }

    fun releaseAll() {
        cancelFade()
        releasePlayer(outgoing)
        releasePlayer(current)
        outgoing = null
        current = null
        currentUrl = null
    }

    private fun pauseAll() {
        cancelFade()
        val incoming = current
        incoming?.pause()
        incoming?.volume = 1f
        releasePlayer(outgoing)
        outgoing = null
    }

    private fun newPlayer(url: String): ExoPlayer {
        val httpFactory = DefaultHttpDataSource.Factory()
            .setUserAgent("IceTray")
            .setAllowCrossProtocolRedirects(true)
        val player = ExoPlayer.Builder(appContext)
            .setMediaSourceFactory(DefaultMediaSourceFactory(httpFactory))
            .build()
        player.setAudioAttributes(mediaAttributes(), false)
        player.setWakeMode(C.WAKE_MODE_NETWORK)
        player.setMediaItem(MediaItem.fromUri(Uri.parse(url)))
        player.addListener(object : Player.Listener {
            override fun onPlaybackStateChanged(playbackState: Int) {
                if (playbackState == Player.STATE_READY && current === player && outgoing == null) {
                    notifyReady()
                }
            }

            override fun onPlayerError(error: PlaybackException) {
                Log.e(TAG, "ExoPlayer error for $url", error)
            }
        })
        player.prepare()
        return player
    }

    private fun startFade(from: ExoPlayer, to: ExoPlayer, durationMs: Int) {
        if (from == to) {
            to.volume = 1f
            notifyReady()
            return
        }
        val generation = ++fadeGeneration
        val startedAt = SystemClock.uptimeMillis()
        val tick = object : Runnable {
            override fun run() {
                if (generation != fadeGeneration || current !== to) {
                    return
                }
                val t = ((SystemClock.uptimeMillis() - startedAt).toFloat() / durationMs.toFloat()).coerceIn(0f, 1f)
                from.volume = cos(0.5 * Math.PI * t).toFloat()
                to.volume = sin(0.5 * Math.PI * t).toFloat()
                if (t < 1f) {
                    mainHandler.postDelayed(this, 32L)
                    return
                }
                fadeRunnable = null
                to.volume = 1f
                if (outgoing === from) {
                    outgoing = null
                }
                releasePlayer(from)
                notifyReady()
            }
        }
        fadeRunnable = tick
        mainHandler.post(tick)
    }

    private fun notifyReady() {
        mainHandler.post { NativeBridge.nativeExoReady() }
    }

    private fun cancelFade() {
        fadeGeneration++
        fadeRunnable?.let { mainHandler.removeCallbacks(it) }
        fadeRunnable = null
    }

    private fun releasePlayer(player: ExoPlayer?) {
        if (player == null) {
            return
        }
        player.stop()
        player.release()
    }

    private fun mediaAttributes(): AudioAttributes {
        return AudioAttributes.Builder()
            .setUsage(C.USAGE_MEDIA)
            .setContentType(C.AUDIO_CONTENT_TYPE_MUSIC)
            .build()
    }

    private companion object {
        const val TAG = "IceTrayExo"
    }
}
