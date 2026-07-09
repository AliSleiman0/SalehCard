package com.example.mobilebridgev2.ussd

import android.app.Activity
import android.app.KeyguardManager
import android.os.Bundle
import android.view.WindowManager
import com.example.mobilebridgev2.net.BridgeReporter

/**
 * Transparent, no-UI trampoline that exists for two reasons the bridge service can't do from
 * the background:
 *  1) Wake the screen and dismiss the (insecure) keyguard so the system USSD dialog actually
 *     renders — with the screen off it never draws and the AccessibilityService has nothing to
 *     drive. The dedicated bridge phone has NO secure lock, so requestDismissKeyguard clears it.
 *  2) Provide a real Activity context to fire ACTION_CALL, and a foreground window the USSD
 *     dialog stacks on top of, keeping the screen alive (FLAG_KEEP_SCREEN_ON) for the whole
 *     session.
 *
 * It stays alive until [UssdSessionCoordinator] clears the session (via the onFinish hook),
 * then finishes itself.
 */
class UssdSessionActivity : Activity() {

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // Turn the screen on, show over the lockscreen, and keep it on for the session.
        setShowWhenLocked(true)
        setTurnScreenOn(true)
        window.addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON)
        (getSystemService(KeyguardManager::class.java))?.requestDismissKeyguard(this, null)

        // Wire finish() so the coordinator can dismiss us when the session ends.
        UssdSessionCoordinator.activeSession?.onFinish = { runOnUiThread { finishAndRemoveTask() } }

        // Only dial on first creation — never re-dial on a recreation (rotation/config change).
        if (savedInstanceState == null) {
            startUssdDial()
        }
    }

    private fun startUssdDial() {
        val session = UssdSessionCoordinator.activeSession
        if (session == null) {
            // No session in flight — nothing to do (e.g. already resolved/timed out).
            finishAndRemoveTask()
            return
        }

        val subId = intent.getIntExtra(EXTRA_SUB_ID, -1)
        val ussd = intent.getStringExtra(EXTRA_USSD)
        if (subId < 0 || ussd.isNullOrBlank()) {
            UssdSessionCoordinator.fail("Missing dial extras")
            return
        }

        val dialIntent = UssdDialer.buildDialIntent(this, subId, ussd)
        if (dialIntent == null) {
            UssdSessionCoordinator.fail("Could not resolve Alfa SIM phone account")
            return
        }

        try {
            session.phase = UssdSessionCoordinator.Phase.AWAITING_PROMPT
            startActivity(dialIntent)
        } catch (e: Exception) {
            BridgeReporter.log("WARN", "ACTION_CALL failed: ${e.message}")
            UssdSessionCoordinator.fail("Dial failed: ${e.message}")
        }
    }

    companion object {
        const val EXTRA_SUB_ID = "com.example.mobilebridgev2.ussd.SUB_ID"
        const val EXTRA_USSD = "com.example.mobilebridgev2.ussd.USSD"
    }
}
