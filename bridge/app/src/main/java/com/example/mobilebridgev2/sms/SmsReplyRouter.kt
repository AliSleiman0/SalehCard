package com.example.mobilebridgev2.sms

import kotlinx.coroutines.CompletableDeferred

object SmsReplyRouter {

    private data class PendingReply(
        val expectedSenders: List<String>,
        val deferred: CompletableDeferred<SmsReply>
    )

    private var pendingReply: PendingReply? = null

    fun prepareReplyWait(expectedSenders: List<String>): CompletableDeferred<SmsReply> {
        val deferred = CompletableDeferred<SmsReply>()

        synchronized(this) {
            check(pendingReply == null) {
                "A provider SMS reply is already being awaited"
            }
            pendingReply = PendingReply(
                expectedSenders = expectedSenders.map { normalize(it) },
                deferred = deferred
            )
        }

        deferred.invokeOnCompletion {
            clearReplyWait(deferred)
        }

        return deferred
    }

    fun clearReplyWait(deferred: CompletableDeferred<SmsReply>) {
        synchronized(this) {
            if (pendingReply?.deferred == deferred) {
                pendingReply = null
            }
        }
    }

    fun onSmsReceived(sender: String, body: String) {
        val normalizedSender = normalize(sender)

        val currentPending = synchronized(this) {
            pendingReply
        } ?: return

        val matchesSender =
            currentPending.expectedSenders.isEmpty() ||
                    currentPending.expectedSenders.any { expected ->
                        normalizedSender.contains(expected)
                    }

        if (!matchesSender) return

        val completed = currentPending.deferred.complete(
            SmsReply(
                sender = sender,
                body = body
            )
        )

        if (completed) {
            synchronized(this) {
                if (pendingReply?.deferred == currentPending.deferred) {
                    pendingReply = null
                }
            }
        }
    }

    fun cancelPendingReply() {
        val deferred = synchronized(this) {
            val current = pendingReply?.deferred
            pendingReply = null
            current
        }

        deferred?.cancel()
    }

    private fun normalize(value: String): String {
        return value
            .lowercase()
            .replace(" ", "")
            .replace("-", "")
    }
}