package com.example.mobilebridgev2.ussd

import android.accessibilityservice.AccessibilityService
import android.os.Bundle
import android.view.accessibility.AccessibilityEvent
import android.view.accessibility.AccessibilityNodeInfo
import com.example.mobilebridgev2.net.BridgeReporter

/**
 * Drives the interactive Alfa recharge USSD dialog. It is globally enabled but stays completely
 * inert unless [UssdSessionCoordinator.activeSession] is non-null — that gate is what scopes all
 * of this behaviour to Alfa recharge only (Touch/balance/SMS never open an active session).
 *
 * For each event while a session is active it:
 *  - reads the dialog text (the operator prompt / result),
 *  - if the dialog has an input field, types the confirm digit and taps Send (multi-step safe),
 *  - if the dialog is a terminal result (text, no input), reports it back to the coroutine.
 *
 * Node classes / button labels vary by OEM + locale, so matching is deliberately lenient and
 * every dialog is logged (logcat) to make on-device bring-up quick.
 */
class AlfaUssdAccessibilityService : AccessibilityService() {

    override fun onAccessibilityEvent(event: AccessibilityEvent?) {
        val session = UssdSessionCoordinator.activeSession ?: return
        if (event == null) return
        when (event.eventType) {
            AccessibilityEvent.TYPE_WINDOW_STATE_CHANGED,
            AccessibilityEvent.TYPE_WINDOW_CONTENT_CHANGED -> Unit
            else -> return
        }

        val root = bestRoot(event) ?: return
        val promptText = collectText(root).trim()
        val editText = findEditable(root)

        // Ignore the app's own transparent trampoline window and empty/progress frames.
        if (promptText.isEmpty() && editText == null) return

        BridgeReporter.log(
            "INFO",
            "USSD dialog pkg=${event.packageName} hasInput=${editText != null} text=\"$promptText\"",
        )

        if (editText != null) {
            handlePrompt(session, root, editText, promptText)
        } else {
            handleMaybeTerminal(promptText)
        }
    }

    private fun handlePrompt(
        session: UssdSessionCoordinator.Session,
        root: AccessibilityNodeInfo,
        editText: AccessibilityNodeInfo,
        promptText: String,
    ) {
        if (session.step >= MAX_STEPS) {
            UssdSessionCoordinator.fail("Too many interactive prompts (>$MAX_STEPS)")
            return
        }

        val digit = session.confirmDigits.getOrNull(session.step)
            ?: parseConfirmDigit(promptText)
            ?: DEFAULT_CONFIRM_DIGIT

        // Always (re)fill the field — cheap and idempotent — but only commit the step once the
        // Send button is actually present and clicked, so a prompt that arrives before its button
        // is laid out just retries on the next content-changed event.
        setText(editText, digit)

        val sendButton = findSendButton(root)
        if (sendButton == null) {
            BridgeReporter.log("WARN", "Prompt shown but no Send button yet; awaiting next event")
            return
        }

        val signature = "prompt:${session.step}:$promptText"
        if (session.lastHandledSignature == signature) return
        session.lastHandledSignature = signature

        val clicked = clickableSelfOrAncestor(sendButton)?.performAction(AccessibilityNodeInfo.ACTION_CLICK) == true
        BridgeReporter.log("INFO", "Answered prompt step=${session.step} digit=$digit sent=$clicked")
        session.step += 1
        session.phase = UssdSessionCoordinator.Phase.AWAITING_TERMINAL
    }

    private fun handleMaybeTerminal(promptText: String) {
        val lower = promptText.lowercase()
        // Transient progress frames are NOT terminal — keep waiting for the real result.
        if (promptText.isBlank() || IGNORE_PATTERNS.any { it in lower }) return
        UssdSessionCoordinator.complete(promptText)
    }

    override fun onInterrupt() {}

    // ---- node helpers -------------------------------------------------------------------

    /** Prefer the active window, but scan all windows for the one actually holding the dialog. */
    private fun bestRoot(event: AccessibilityEvent): AccessibilityNodeInfo? {
        val candidates = mutableListOf<AccessibilityNodeInfo>()
        rootInActiveWindow?.let { candidates.add(it) }
        try {
            windows?.forEach { w -> w.root?.let { candidates.add(it) } }
        } catch (_: Exception) { /* windows may be unavailable */ }
        event.source?.let { candidates.add(it) }
        if (candidates.isEmpty()) return null
        // Choose the node subtree with an editable field, else the one with the most text.
        return candidates.firstOrNull { findEditable(it) != null }
            ?: candidates.maxByOrNull { collectText(it).length }
    }

    private fun collectText(node: AccessibilityNodeInfo?): String {
        if (node == null) return ""
        val sb = StringBuilder()
        fun walk(n: AccessibilityNodeInfo?) {
            if (n == null) return
            n.text?.let { if (it.isNotBlank()) sb.append(it).append(' ') }
            n.contentDescription?.let { if (it.isNotBlank()) sb.append(it).append(' ') }
            for (i in 0 until n.childCount) walk(n.getChild(i))
        }
        walk(node)
        return sb.toString().replace(Regex("\\s+"), " ").trim()
    }

    private fun findEditable(node: AccessibilityNodeInfo?): AccessibilityNodeInfo? {
        if (node == null) return null
        if (node.isEditable || node.className?.toString()?.contains("EditText", true) == true) return node
        for (i in 0 until node.childCount) {
            findEditable(node.getChild(i))?.let { return it }
        }
        return null
    }

    private fun findSendButton(node: AccessibilityNodeInfo?): AccessibilityNodeInfo? {
        if (node == null) return null
        val clickables = mutableListOf<AccessibilityNodeInfo>()
        fun walk(n: AccessibilityNodeInfo?) {
            if (n == null) return
            val label = (n.text?.toString() ?: "") + " " + (n.contentDescription?.toString() ?: "")
            if (SEND_LABELS.any { label.contains(it, ignoreCase = true) }) {
                clickables.add(0, n) // preferred: label match first
            } else if (n.isClickable && (n.className?.toString()?.contains("Button", true) == true)) {
                clickables.add(n) // fallback candidate
            }
            for (i in 0 until n.childCount) walk(n.getChild(i))
        }
        walk(node)
        // Avoid picking a Cancel/back/negative button.
        return clickables.firstOrNull { c ->
            val label = (c.text?.toString() ?: "") + " " + (c.contentDescription?.toString() ?: "")
            NEGATIVE_LABELS.none { label.contains(it, ignoreCase = true) }
        } ?: clickables.firstOrNull()
    }

    private fun clickableSelfOrAncestor(node: AccessibilityNodeInfo?): AccessibilityNodeInfo? {
        var n = node
        var guard = 0
        while (n != null && guard < 8) {
            if (n.isClickable) return n
            n = n.parent
            guard++
        }
        return node
    }

    private fun setText(editText: AccessibilityNodeInfo, value: String) {
        val args = Bundle().apply {
            putCharSequence(AccessibilityNodeInfo.ACTION_ARGUMENT_SET_TEXT_CHARSEQUENCE, value)
        }
        editText.performAction(AccessibilityNodeInfo.ACTION_SET_TEXT, args)
    }

    private fun parseConfirmDigit(promptText: String): String? =
        Regex("""press\s+(\d)""", RegexOption.IGNORE_CASE).find(promptText)?.groupValues?.get(1)
            ?: Regex("""(\d)\s*[-.)]\s*confirm""", RegexOption.IGNORE_CASE)
                .find(promptText)?.groupValues?.get(1)

    companion object {
        private const val MAX_STEPS = 3
        private const val DEFAULT_CONFIRM_DIGIT = "1"
        // Positive/submit labels across EN + AR (Alfa Lebanon locale may be either).
        private val SEND_LABELS = listOf("send", "ok", "yes", "confirm", "submit", "إرسال", "موافق", "نعم", "تأكيد")
        private val NEGATIVE_LABELS = listOf("cancel", "back", "no", "إلغاء", "رجوع", "لا")
        private val IGNORE_PATTERNS = listOf(
            "running", "ussd code", "please wait", "loading", "connecting", "sending", "processing",
        )
    }
}
