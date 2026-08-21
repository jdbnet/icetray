package uk.co.jdbnet.icetray.playback

import android.content.ComponentName
import android.content.Context
import android.content.Intent
import android.content.ServiceConnection
import android.os.IBinder
import androidx.media3.common.Player
import androidx.media3.session.MediaSession
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import uk.co.jdbnet.icetray.data.NowPlaying
import uk.co.jdbnet.icetray.data.PlaybackState
import uk.co.jdbnet.icetray.data.StreamView

object PlaybackController {
    private var service: PlaybackService? = null
    private var bound = false
    private var pendingPlay: PendingPlay? = null

    private val _playbackState = MutableStateFlow(PlaybackState())
    val playbackState: StateFlow<PlaybackState> = _playbackState.asStateFlow()

    private val _nowPlaying = MutableStateFlow(NowPlaying())
    val nowPlaying: StateFlow<NowPlaying> = _nowPlaying.asStateFlow()

    private var player: Player? = null
    private var mediaSession: MediaSession? = null

    private data class PendingPlay(val stream: StreamView)

    private val connection = object : ServiceConnection {
        override fun onServiceConnected(name: ComponentName?, binder: IBinder?) {
            bound = true
            pendingPlay?.let { pending ->
                service?.playStream(pending.stream)
                pendingPlay = null
            }
        }

        override fun onServiceDisconnected(name: ComponentName?) {
            bound = false
            service = null
        }
    }

    fun attachService(service: PlaybackService) {
        this.service = service
    }

    fun detachService() {
        service = null
        player = null
        mediaSession = null
    }

    fun onPlayerReady(exoPlayer: Player, session: MediaSession) {
        player = exoPlayer
        mediaSession = session
    }

    fun bind(context: Context) {
        if (bound) return
        context.applicationContext.bindService(
            Intent(context, PlaybackService::class.java),
            connection,
            Context.BIND_AUTO_CREATE,
        )
    }

    fun unbind(context: Context) {
        if (!bound) return
        runCatching {
            context.applicationContext.unbindService(connection)
        }
        bound = false
        service = null
    }

    fun play(context: Context, stream: StreamView) {
        val intent = PlaybackService.intentForStream(context, stream)
        context.startForegroundService(intent)
        bind(context)
        service?.playStream(stream) ?: run { pendingPlay = PendingPlay(stream) }
    }

    fun pause() {
        service?.pause()
    }

    fun resume() {
        service?.resume()
    }

    fun stop(context: Context) {
        service?.stopPlayback()
        unbind(context)
        updatePlaybackState(PlaybackState())
        updateNowPlaying(NowPlaying())
    }

    fun updatePlaybackState(state: PlaybackState) {
        _playbackState.value = state
    }

    fun updateNowPlaying(nowPlaying: NowPlaying) {
        _nowPlaying.value = nowPlaying
    }
}
