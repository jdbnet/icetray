package uk.co.jdbnet.icetray.playback

import android.content.Context
import androidx.annotation.StringRes
import androidx.media3.common.Player
import androidx.media3.common.util.UnstableApi
import androidx.media3.session.CommandButton
import androidx.media3.session.DefaultMediaNotificationProvider
import androidx.media3.session.MediaSession
import com.google.common.collect.ImmutableList
import uk.co.jdbnet.icetray.R

@UnstableApi
class IceTrayMediaNotificationProvider(
    private val appContext: Context,
    channelId: String,
    @StringRes channelNameResId: Int,
) : DefaultMediaNotificationProvider(
    appContext,
    { DEFAULT_NOTIFICATION_ID },
    channelId,
    channelNameResId,
) {
    override fun getMediaButtons(
        session: MediaSession,
        playerCommands: Player.Commands,
        mediaButtonPreferences: ImmutableList<CommandButton>,
        showPauseButton: Boolean,
    ): ImmutableList<CommandButton> {
        val buttons = ImmutableList.Builder<CommandButton>()
        if (playerCommands.contains(Player.COMMAND_PLAY_PAUSE)) {
            val icon = if (showPauseButton) R.drawable.ic_media_pause else R.drawable.ic_media_play
            buttons.add(
                CommandButton.Builder(icon)
                    .setPlayerCommand(Player.COMMAND_PLAY_PAUSE)
                    .setDisplayName(
                        appContext.getString(
                            if (showPauseButton) R.string.pause else R.string.play,
                        ),
                    )
                    .build(),
            )
        }
        if (playerCommands.contains(Player.COMMAND_STOP)) {
            buttons.add(
                CommandButton.Builder(R.drawable.ic_media_stop)
                    .setPlayerCommand(Player.COMMAND_STOP)
                    .setDisplayName(appContext.getString(R.string.stop))
                    .build(),
            )
        }
        return buttons.build()
    }
}
