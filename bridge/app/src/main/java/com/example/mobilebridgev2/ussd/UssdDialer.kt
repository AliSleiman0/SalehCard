package com.example.mobilebridgev2.ussd

import android.annotation.SuppressLint
import android.content.Context
import android.net.Uri
import android.os.Bundle
import android.telecom.PhoneAccountHandle
import android.telecom.TelecomManager
import com.example.mobilebridgev2.ProviderStore
import com.example.mobilebridgev2.net.BridgeReporter

/**
 * Places the USSD call that surfaces the system USSD dialog on a SPECIFIC dual-SIM subscription.
 * Targeting the right SIM matters: dialing a recharge USSD on the wrong SIM mis-charges the other
 * line, so we resolve the Alfa SIM's PhoneAccountHandle explicitly and refuse to fall back to the
 * default SIM.
 *
 * Uses [TelecomManager.placeCall] rather than an `ACTION_CALL` intent: CALL is placed by the
 * Telecom system service, not by an Activity in the dialer package, so intent resolution either
 * pops a "Complete action using…" chooser (multiple tel handlers) or fails with "no Activity
 * found" when pinned to the dialer package. placeCall goes straight through Telecom — no chooser,
 * no Activity resolution — and honours the PhoneAccountHandle.
 */
object UssdDialer {

    /**
     * Places [ussd] as a call on the SIM identified by [subId], surfacing the USSD dialog for the
     * accessibility service to drive. Returns false if the SIM's PhoneAccountHandle can't be
     * resolved or the call can't be placed (caller must then fail the command).
     */
    @SuppressLint("MissingPermission")
    fun placeUssdCall(context: Context, subId: Int, ussd: String): Boolean {
        val telecom = context.getSystemService(TelecomManager::class.java) ?: return false
        val handle = resolvePhoneAccountHandle(context, subId) ?: return false
        // Build the tel: URI via fromParts so '#' is preserved in the scheme-specific part
        // (Uri.parse would treat '#' as a fragment delimiter and drop the USSD tail).
        val uri = Uri.fromParts("tel", ussd, null)
        val extras = Bundle().apply {
            putParcelable(TelecomManager.EXTRA_PHONE_ACCOUNT_HANDLE, handle)
        }
        return try {
            telecom.placeCall(uri, extras)
            true
        } catch (e: Exception) {
            BridgeReporter.log("WARN", "placeCall failed: ${e.message}")
            false
        }
    }

    /**
     * Maps a subscription id to its call-capable [PhoneAccountHandle]. Public APIs before
     * Android 14 don't expose subId→handle directly, so we use layered heuristics and log every
     * candidate for on-device verification.
     */
    @SuppressLint("MissingPermission")
    fun resolvePhoneAccountHandle(context: Context, subId: Int): PhoneAccountHandle? {
        val telecom = context.getSystemService(TelecomManager::class.java) ?: return null
        val handles: List<PhoneAccountHandle> = try {
            telecom.callCapablePhoneAccounts
        } catch (e: SecurityException) {
            BridgeReporter.log("WARN", "callCapablePhoneAccounts denied: ${e.message}")
            return null
        }

        if (handles.isEmpty()) {
            BridgeReporter.log("WARN", "No call-capable phone accounts found")
            return null
        }

        // Debug aid (logcat only): dump every handle so the id/label mapping can be verified.
        handles.forEach { h ->
            val label = runCatching { telecom.getPhoneAccount(h)?.label?.toString() }.getOrNull()
            BridgeReporter.log("INFO", "PhoneAccount id='${h.id}' label='$label'")
        }

        // 1) Primary heuristic: the account id equals the subscription id string on most devices.
        handles.firstOrNull { it.id == subId.toString() }?.let { return it }

        // 2) Match the account label to the Alfa SIM's carrier/display name (same lowercase
        //    "alfa"-contains logic used when SIMs are matched to providers at startup).
        val alfaName = ProviderStore.alfaSim
            ?.let { "${it.carrierName} ${it.displayName}" }
            ?.lowercase()
        handles.firstOrNull { h ->
            val label = runCatching { telecom.getPhoneAccount(h)?.label?.toString() }
                .getOrNull()?.lowercase()
            label != null && ("alfa" in label || (alfaName != null && label.isNotBlank() && alfaName.contains(label)))
        }?.let { return it }

        // 3) Two accounts, one is clearly Touch → pick the other.
        if (handles.size == 2) {
            val nonTouch = handles.firstOrNull { h ->
                val label = runCatching { telecom.getPhoneAccount(h)?.label?.toString() }
                    .getOrNull()?.lowercase() ?: ""
                "touch" !in label && "mtc" !in label
            }
            if (nonTouch != null) return nonTouch
        }

        BridgeReporter.log("WARN", "Could not resolve Alfa PhoneAccountHandle for subId=$subId")
        return null
    }
}
