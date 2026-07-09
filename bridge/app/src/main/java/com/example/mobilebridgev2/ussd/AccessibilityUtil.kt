package com.example.mobilebridgev2.ussd

import android.accessibilityservice.AccessibilityService
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.provider.Settings
import android.text.TextUtils

/** Helpers for the special-access permissions the interactive-USSD flow depends on. */
object AccessibilityUtil {

    /**
     * True if [service] is currently enabled in Settings → Accessibility. Parses the
     * colon-separated `ENABLED_ACCESSIBILITY_SERVICES` secure setting for our
     * `package/serviceClass` component and confirms accessibility is globally on.
     */
    fun isServiceEnabled(
        context: Context,
        service: Class<out AccessibilityService>,
    ): Boolean {
        val enabledGlobally = Settings.Secure.getInt(
            context.contentResolver,
            Settings.Secure.ACCESSIBILITY_ENABLED,
            0,
        ) == 1
        if (!enabledGlobally) return false

        val expected = "${context.packageName}/${service.name}"
        val enabledServices = Settings.Secure.getString(
            context.contentResolver,
            Settings.Secure.ENABLED_ACCESSIBILITY_SERVICES,
        ) ?: return false

        val splitter = TextUtils.SimpleStringSplitter(':')
        splitter.setString(enabledServices)
        for (component in splitter) {
            // Match either the fully-qualified name or the short `.Class` form Android may store.
            if (component.equals(expected, ignoreCase = true) ||
                component.endsWith("/${service.name}", ignoreCase = true) ||
                component.endsWith("/${service.simpleName}", ignoreCase = true)
            ) {
                return true
            }
        }
        return false
    }

    /** True if the app can start activities from the background / draw overlays (BAL exemption). */
    fun overlayGranted(context: Context): Boolean = Settings.canDrawOverlays(context)

    fun openAccessibilitySettings(): Intent =
        Intent(Settings.ACTION_ACCESSIBILITY_SETTINGS)
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)

    fun openOverlaySettings(context: Context): Intent =
        Intent(
            Settings.ACTION_MANAGE_OVERLAY_PERMISSION,
            Uri.parse("package:${context.packageName}"),
        ).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
}
