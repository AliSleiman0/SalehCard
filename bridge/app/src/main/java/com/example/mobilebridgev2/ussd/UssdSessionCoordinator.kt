package com.example.mobilebridgev2.ussd

import kotlinx.coroutines.CompletableDeferred
import java.util.concurrent.atomic.AtomicReference

/**
 * Process-wide bridge between the [AlfaUssdAccessibilityService] (which runs on the main
 * thread and drives the system USSD dialog) and the suspend `rechargeAlfa` coroutine (which
 * runs on Dispatchers.IO and awaits the outcome).
 *
 * Only ONE interactive USSD session can be active at a time — this matches the bridge's
 * existing "one USSD at a time" invariant (a single serial command worker, and a single
 * global pending SMS reply in SmsReplyRouter). The accessibility service is globally enabled
 * but stays completely inert unless [activeSession] is non-null, which is how the whole
 * interactive-USSD behaviour is scoped to Alfa recharge only.
 */
object UssdSessionCoordinator {

    /** What the accessibility service (or a timeout) reports back to the coroutine. */
    sealed interface Outcome {
        /** The session reached a terminal dialog; [rawReply] is the operator's message text. */
        data class Terminal(val rawReply: String) : Outcome
        /** No terminal dialog within the timeout window. */
        data object Timeout : Outcome
        /** Could not drive the session (dialog never appeared, node not found, etc.). */
        data class Error(val reason: String) : Outcome
    }

    enum class Phase { DIALING, AWAITING_PROMPT, RESPONDING, AWAITING_TERMINAL, DONE }

    class Session(
        val commandId: String,
        /** Digits to type at each interactive prompt, in order. Default: a single "1". */
        val confirmDigits: List<String>,
        val deferred: CompletableDeferred<Outcome>,
    ) {
        @Volatile var phase: Phase = Phase.DIALING
        @Volatile var step: Int = 0
        /** De-dupe key so repeated WINDOW_CONTENT_CHANGED events don't double-submit. */
        @Volatile var lastHandledSignature: String? = null
        /** Invoked once when the session is cleared — finishes the trampoline Activity. */
        @Volatile var onFinish: (() -> Unit)? = null
    }

    private val ref = AtomicReference<Session?>(null)

    val activeSession: Session? get() = ref.get()

    /**
     * Registers [session] as the active one. Any leftover session (e.g. from a prior crash) is
     * resolved as an error first so it can never wedge the serial worker. Always succeeds.
     */
    fun begin(session: Session) {
        val previous = ref.getAndSet(session)
        if (previous != null && previous.deferred.isActive) {
            previous.deferred.complete(Outcome.Error("superseded by a new session"))
        }
    }

    fun complete(rawReply: String) = resolve(Outcome.Terminal(rawReply))

    fun fail(reason: String) = resolve(Outcome.Error(reason))

    fun timeout() = resolve(Outcome.Timeout)

    private fun resolve(outcome: Outcome) {
        val session = ref.get() ?: return
        session.phase = Phase.DONE
        // Idempotent: a late accessibility event after a timeout must not crash or re-resolve.
        if (session.deferred.isActive) session.deferred.complete(outcome)
        clear(session)
    }

    /** Clears the active session (if it is still [session]) and finishes the trampoline. */
    fun clear(session: Session) {
        if (ref.compareAndSet(session, null)) {
            val finish = session.onFinish
            session.onFinish = null
            finish?.invoke()
        }
    }
}
