package uk.co.jdbnet.icetray.playback

import android.content.Context
import android.os.Bundle
import androidx.annotation.StringRes
import androidx.core.app.NotificationCompat
import androidx.media3.common.Player
import androidx.media3.common.util.UnstableApi
import androidx.media3.session.CommandButton
import androidx.media3.session.DefaultMediaNotificationProvider
import androidx.media3.session.DefaultMediaNotificationProvider.DEFAULT_NOTIFICATION_ID
import androidx.media3.session.MediaNotification
import androidx.media3.session.MediaSession
import com.google.common.collect.ImmutableList
import com.wails.app.R

@UnstableApi
class IceTrayMediaNotificationProvider(
    private val appContext: Context,
    private val channelId: String,
    @StringRes channelNameResId: Int,
) : MediaNotification.Provider {

    private val delegate: DefaultMediaNotificationProvider =
        object : DefaultMediaNotificationProvider(
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
                return buildIceTrayMediaButtons(
                    playLabel = appContext.getString(R.string.play),
                    pauseLabel = appContext.getString(R.string.pause),
                    stopLabel = appContext.getString(R.string.stop),
                    includePlayPause = playerCommands.contains(Player.COMMAND_PLAY_PAUSE),
                    includeStop = playerCommands.contains(Player.COMMAND_STOP),
                    showPauseButton = showPauseButton,
                )
            }
        }

    fun setSmallIcon(iconResId: Int) {
        delegate.setSmallIcon(iconResId)
    }

    override fun createNotification(
        mediaSession: MediaSession,
        mediaButtonPreferences: ImmutableList<CommandButton>,
        actionFactory: MediaNotification.ActionFactory,
        onNotificationChangedCallback: MediaNotification.Provider.Callback,
    ): MediaNotification {
        val result = delegate.createNotification(
            mediaSession,
            mediaButtonPreferences,
            actionFactory,
            onNotificationChangedCallback,
        )
        return silencePlaybackNotification(appContext, result)
    }

    override fun handleCustomCommand(
        session: MediaSession,
        action: String,
        extras: Bundle,
    ): Boolean {
        return delegate.handleCustomCommand(session, action, extras)
    }

    override fun getNotificationChannelInfo(): MediaNotification.Provider.NotificationChannelInfo {
        return delegate.notificationChannelInfo
    }
}

@UnstableApi
internal fun silencePlaybackNotification(
    context: Context,
    result: MediaNotification,
): MediaNotification {
    val quiet = NotificationCompat.Builder(context, result.notification)
        .setSilent(true)
        .setOnlyAlertOnce(true)
        .build()
    return MediaNotification(result.notificationId, quiet)
}

@UnstableApi
internal data class IceTrayMediaButtonSpec(
    val icon: Int,
    val customIconResId: Int,
    val playerCommand: Int,
    val displayName: String,
)

@UnstableApi
internal fun iceTrayMediaButtonSpecs(
    playLabel: String,
    pauseLabel: String,
    stopLabel: String,
    includePlayPause: Boolean,
    includeStop: Boolean,
    showPauseButton: Boolean,
): List<IceTrayMediaButtonSpec> {
    val specs = mutableListOf<IceTrayMediaButtonSpec>()
    if (includePlayPause) {
        specs.add(
            IceTrayMediaButtonSpec(
                icon = if (showPauseButton) CommandButton.ICON_PAUSE else CommandButton.ICON_PLAY,
                customIconResId = if (showPauseButton) R.drawable.ic_media_pause else R.drawable.ic_media_play,
                playerCommand = Player.COMMAND_PLAY_PAUSE,
                displayName = if (showPauseButton) pauseLabel else playLabel,
            ),
        )
    }
    if (includeStop) {
        specs.add(
            IceTrayMediaButtonSpec(
                icon = CommandButton.ICON_STOP,
                customIconResId = R.drawable.ic_media_stop,
                playerCommand = Player.COMMAND_STOP,
                displayName = stopLabel,
            ),
        )
    }
    return specs
}

@UnstableApi
internal fun buildIceTrayMediaButtons(
    playLabel: String,
    pauseLabel: String,
    stopLabel: String,
    includePlayPause: Boolean,
    includeStop: Boolean,
    showPauseButton: Boolean,
): ImmutableList<CommandButton> {
    val buttons = ImmutableList.Builder<CommandButton>()
    for (spec in iceTrayMediaButtonSpecs(
        playLabel = playLabel,
        pauseLabel = pauseLabel,
        stopLabel = stopLabel,
        includePlayPause = includePlayPause,
        includeStop = includeStop,
        showPauseButton = showPauseButton,
    )) {
        buttons.add(
            CommandButton.Builder(spec.icon)
                .setCustomIconResId(spec.customIconResId)
                .setPlayerCommand(spec.playerCommand)
                .setDisplayName(spec.displayName)
                .build(),
        )
    }
    return buttons.build()
}
