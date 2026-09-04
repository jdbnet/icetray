package uk.co.jdbnet.icetray.playback

import android.net.Uri
import android.os.Looper
import androidx.media3.common.C
import androidx.media3.common.MediaItem
import androidx.media3.common.MediaMetadata
import androidx.media3.common.Player
import androidx.media3.common.SimpleBasePlayer
import androidx.media3.common.util.UnstableApi
import com.google.common.util.concurrent.Futures
import com.google.common.util.concurrent.ListenableFuture
import uk.co.jdbnet.icetray.NativeBridge
import java.io.File

@UnstableApi
class GoBackingPlayer(looper: Looper) : SimpleBasePlayer(looper) {
    private var playing = false
    private var paused = false
    private var title = "IceTray"
    private var artist = "IceTray"
    private var artworkUri: Uri? = null
    private var streamId = ""

    fun applySession(
        playing: Boolean,
        paused: Boolean,
        title: String,
        artist: String,
        artworkPath: String?,
        streamId: String,
    ) {
        this.playing = playing
        this.paused = paused
        this.title = title.ifBlank { "IceTray" }
        this.artist = artist.ifBlank { "IceTray" }
        this.artworkUri = artworkPath?.takeIf { it.isNotBlank() }?.let { Uri.fromFile(File(it)) }
        this.streamId = streamId
        invalidateState()
    }

    override fun getState(): State {
        val playbackState = if (playing || paused) Player.STATE_READY else Player.STATE_IDLE
        val metadata = MediaMetadata.Builder()
            .setTitle(title)
            .setDisplayTitle(title)
            .setArtist(artist)
            .setAlbumTitle(artist)
            .apply { artworkUri?.let { setArtworkUri(it) } }
            .build()
        val mediaId = streamId.ifBlank { "icetray" }
        val item = MediaItem.Builder()
            .setMediaId(mediaId)
            .setMediaMetadata(metadata)
            .build()
        val itemData = MediaItemData.Builder(mediaId)
            .setMediaItem(item)
            .setMediaMetadata(metadata)
            .setDurationUs(C.TIME_UNSET)
            .setIsSeekable(false)
            .setIsDynamic(true)
            .build()
        val commands = Player.Commands.Builder()
            .addAll(
                Player.COMMAND_PLAY_PAUSE,
                Player.COMMAND_STOP,
                Player.COMMAND_GET_CURRENT_MEDIA_ITEM,
                Player.COMMAND_GET_METADATA,
                Player.COMMAND_GET_TIMELINE,
            )
            .build()
        val builder = State.Builder()
            .setAvailableCommands(commands)
            .setPlayWhenReady(playing, Player.PLAY_WHEN_READY_CHANGE_REASON_REMOTE)
            .setPlaybackState(playbackState)
        if (playing || paused) {
            builder.setPlaylist(listOf(itemData))
        }
        return builder.build()
    }

    override fun handleSetPlayWhenReady(playWhenReady: Boolean): ListenableFuture<*> {
        val alreadyPlaying = playing && !paused
        val alreadyPaused = paused
        if (playWhenReady && alreadyPlaying) {
            return Futures.immediateVoidFuture()
        }
        if (!playWhenReady && (alreadyPaused || !playing)) {
            return Futures.immediateVoidFuture()
        }
        if (playWhenReady) {
            if (paused) {
                NativeBridge.nativeResume()
            } else {
                NativeBridge.nativePlayLast()
            }
        } else {
            NativeBridge.nativePause()
        }
        return Futures.immediateVoidFuture()
    }

    override fun handleStop(): ListenableFuture<*> {
        NativeBridge.nativeStop()
        return Futures.immediateVoidFuture()
    }
}
