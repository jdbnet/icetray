package uk.co.jdbnet.icetray

import android.Manifest
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.content.ContextCompat
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import uk.co.jdbnet.icetray.data.StreamView

class BootReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent?) {
        if (intent?.action != Intent.ACTION_BOOT_COMPLETED) return
        val pending = goAsync()
        val app = context.applicationContext as IceTrayApp
        CoroutineScope(Dispatchers.IO).launch {
            try {
                val config = app.configRepository.load()
                if (!config.launchOnLogin || config.lastStreamId.isBlank()) return@launch
                val stream = app.configRepository.getStreams().find { it.id == config.lastStreamId }
                    ?: return@launch
                postResumeNotification(app, stream)
            } finally {
                pending.finish()
            }
        }
    }

    private fun postResumeNotification(context: Context, stream: StreamView) {
        if (!canPostNotifications(context)) return
        ensureChannel(context)

        val launchIntent = Intent(context, MainActivity::class.java).apply {
            action = MainActivity.ACTION_RESUME_LAST_STREAM
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or
                Intent.FLAG_ACTIVITY_CLEAR_TOP or
                Intent.FLAG_ACTIVITY_SINGLE_TOP
            putExtra(MainActivity.EXTRA_RESUME_LAST_STREAM, true)
            putExtra(MainActivity.EXTRA_RESUME_STREAM_ID, stream.id)
        }
        val contentIntent = PendingIntent.getActivity(
            context,
            REQUEST_RESUME,
            launchIntent,
            PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT,
        )

        val notification = NotificationCompat.Builder(context, CHANNEL_ID)
            .setSmallIcon(R.drawable.ic_notification)
            .setContentTitle(context.getString(R.string.boot_resume_title))
            .setContentText(context.getString(R.string.boot_resume_text, stream.name))
            .setCategory(NotificationCompat.CATEGORY_RECOMMENDATION)
            .setAutoCancel(true)
            .setContentIntent(contentIntent)
            .setPriority(NotificationCompat.PRIORITY_DEFAULT)
            .build()

        val manager = context.getSystemService(NotificationManager::class.java) ?: return
        manager.notify(NOTIFICATION_ID, notification)
    }

    private fun canPostNotifications(context: Context): Boolean {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU) return true
        return ContextCompat.checkSelfPermission(
            context,
            Manifest.permission.POST_NOTIFICATIONS,
        ) == PackageManager.PERMISSION_GRANTED
    }

    private fun ensureChannel(context: Context) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val manager = context.getSystemService(NotificationManager::class.java) ?: return
        if (manager.getNotificationChannel(CHANNEL_ID) != null) return
        manager.createNotificationChannel(
            NotificationChannel(
                CHANNEL_ID,
                context.getString(R.string.boot_resume_channel),
                NotificationManager.IMPORTANCE_DEFAULT,
            ).apply {
                description = context.getString(R.string.boot_resume_channel)
                setShowBadge(false)
            },
        )
    }

    companion object {
        private const val CHANNEL_ID = "icetray_boot_resume"
        const val NOTIFICATION_ID = 1001
        private const val REQUEST_RESUME = 2001
    }
}
