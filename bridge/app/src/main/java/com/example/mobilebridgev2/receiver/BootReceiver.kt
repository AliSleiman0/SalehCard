package com.example.mobilebridgev2.receiver

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import androidx.core.content.ContextCompat
import com.example.mobilebridgev2.config.DeviceConfigStore
import com.example.mobilebridgev2.service.BridgeForegroundService

/**
 * Restarts the bridge foreground service after a device reboot — but only when
 * the device has been provisioned (a server URL + token are stored). Without
 * provisioning there is nothing to connect to, so we stay dormant until the
 * operator sets it up in the app.
 */
class BootReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        if (intent.action != Intent.ACTION_BOOT_COMPLETED) return
        DeviceConfigStore.init(context)
        if (!DeviceConfigStore.isProvisioned()) return
        ContextCompat.startForegroundService(
            context,
            Intent(context, BridgeForegroundService::class.java),
        )
    }
}
