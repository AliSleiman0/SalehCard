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
            handleMaybeTerminal(session, root, promptText)
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

        // De-dupe on the prompt CONTENT (not the step counter, which changes after each answer) so
        // the same dialog re-firing WINDOW_CONTENT_CHANGED can't be answered — and Sent — twice.
        // A genuinely different next step has different text and is handled normally.
        val signature = "prompt:$promptText"
        if (session.lastHandledSignature == signature) return
        session.lastHandledSignature = signature

        val clicked = clickableSelfOrAncestor(sendButton)?.performAction(AccessibilityNodeInfo.ACTION_CLICK) == true
        BridgeReporter.log("INFO", "Answered prompt step=${session.step} digit=$digit sent=$clicked")
        session.step += 1
        session.phase = UssdSessionCoordinator.Phase.AWAITING_TERMINAL
    }

    private fun handleMaybeTerminal(
        session: UssdSessionCoordinator.Session,
        root: AccessibilityNodeInfo,
        promptText: String,
    ) {
        // Only accept a terminal reply AFTER we've answered at least one prompt. Before that, any
        // no-input dialog is noise — the "Complete action using" app chooser, a progress frame,
        // etc. — and must NOT be mistaken for the operator's result, or it prematurely ends the
        // session and leaves the real confirm dialog unhandled.
        if (session.phase != UssdSessionCoordinator.Phase.AWAITING_TERMINAL) return
        val lower = promptText.lowercase()
        // Transient progress frames are NOT terminal — keep waiting for the real result.
        if (promptText.isBlank() || IGNORE_PATTERNS.any { it in lower }) return
        // Dismiss the operator's result dialog by clicking its OK/dismiss button (best effort),
        // then report the captured text back to the coroutine.
        findSendButton(root)?.let { clickableSelfOrAncestor(it)?.performAction(AccessibilityNodeInfo.ACTION_CLICK) }
        BridgeReporter.log("INFO", "Terminal reply captured, dismissed: \"$promptText\"")
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

        // Exclude the status bar / system UI and our own trampoline window. The USSD dialog is
        // rendered by the phone/telecom app; the status bar (clock, ringer, signal) otherwise
        // wins the "most text" tiebreak and gets captured as a bogus reply.
        val relevant = candidates.filter { r ->
            val pkg = r.packageName?.toString()
            pkg != "com.android.systemui" && pkg != packageName
        }.ifEmpty { candidates }

        // Prefer the dialog with an input field (the confirm prompt); else the one that has a
        // dialog button AND text (the result dialog); else fall back to the most text.
        return relevant.firstOrNull { findEditable(it) != null }
            ?: relevant.firstOrNull { findSendButton(it) != null && collectText(it).isNotBlank() }
            ?: relevant.maxByOrNull { collectText(it).length }
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
            "complete action using", // Android intent-disambiguation chooser — never a USSD result
        )
    }
}
