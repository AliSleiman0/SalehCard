package com.example.mobilebridgev2

import org.junit.Assert.assertEquals
import org.junit.Test

/**
 * Locks the two live operator balance-reply shapes the device must parse. Touch
 * and Alfa word their balance USSD reply differently; a touch-only parser left
 * every Alfa balance stuck at 0.00 (which then blocks transfer pre-checks).
 */
class BalanceReplyParseTest {

    @Test
    fun parsesTouchExpFormat() {
        // *220# — currency-first, "Exp:", dash date, 2-digit year
        val (balance, validity) = CommandExecutor.getBalanceFromReply("USD 5.00 Exp:10-06-27")
        assertEquals(5.00, balance, 0.0001)
        assertEquals("10-06-27", validity)
    }

    @Test
    fun parsesTouchFourDigitYear() {
        val (balance, validity) = CommandExecutor.getBalanceFromReply("USD 12.50 Exp:10-06-2027")
        assertEquals(12.50, balance, 0.0001)
        assertEquals("10-06-2027", validity)
    }

    @Test
    fun parsesRealTouchReplyWithTrailingDataBundlesAndPromo() {
        // Verbatim live *220# reply: balance line first, then data-bundle lines
        // (each with their own "Exp:" date) and a marketing tail. Must extract the
        // FIRST USD balance (4.26), not a later "GB Exp:" fragment.
        val reply = "USD 4.26 Exp:10-06-27;MI-44GB: 18.46 GB Exp:10-07-26 MST: 19.27 GB " +
                "Exp:15-07-26Get 25GB, 120 Minutes & 120 SMS at only \$14.9/ month! Send WX2 to 1100"
        val (balance, validity) = CommandExecutor.getBalanceFromReply(reply)
        assertEquals(4.26, balance, 0.0001)
        assertEquals("10-06-27", validity)
    }

    @Test
    fun parsesTouchReplyWithIrregularWhitespaceAndNewlines() {
        // The OLD split(" ")+substring parser returned balance=0.00 (a "successful"
        // zero) whenever the USSD text had a double space or newline, because
        // positional indexing shifted off the number. Capture groups are immune —
        // this is the root cause of the Touch balance showing 0.00 / "not enough".
        val doubleSpaced = CommandExecutor.getBalanceFromReply("USD  4.26  Exp: 10-06-27")
        assertEquals(4.26, doubleSpaced.first, 0.0001)
        assertEquals("10-06-27", doubleSpaced.second)

        val newlined = CommandExecutor.getBalanceFromReply("USD 4.26 Exp:10-06-27\nMI-44GB: 18.46 GB")
        assertEquals(4.26, newlined.first, 0.0001)
        assertEquals("10-06-27", newlined.second)
    }

    @Test
    fun parsesAlfaTillFormat() {
        // *11# — amount-first, "till", slash date, 4-digit year
        val (balance, validity) = CommandExecutor.getBalanceFromReply("1.01 USD till 04/09/2026")
        assertEquals(1.01, balance, 0.0001)
        assertEquals("04/09/2026", validity)
    }

    @Test
    fun parsesAlfaReplyEmbeddedInLongerText() {
        val reply = "Dear customer, your balance is 3.50 USD till 04/09/2026. Thank you."
        val (balance, validity) = CommandExecutor.getBalanceFromReply(reply)
        assertEquals(3.50, balance, 0.0001)
        assertEquals("04/09/2026", validity)
    }

    @Test
    fun returnsSentinelOnUnknownReply() {
        val (balance, validity) = CommandExecutor.getBalanceFromReply("Service temporarily unavailable")
        assertEquals(0.0, balance, 0.0001)
        assertEquals("0", validity)
    }
}
