package com.example.mobilebridgev2

import com.example.mobilebridgev2.dto.ConfigurationDTO
import com.google.gson.Gson
import org.junit.Assert.assertEquals
import org.junit.Test

/**
 * Guards the config payload decode. The server sends deviceId as a Mongo ObjectID
 * hex string (d.ID.Hex()); when the DTO typed it as Int, Gson threw
 * NumberFormatException on the whole payload, so the device silently fell back to
 * hardcoded defaults (notably the 20.0 min-balance reserve instead of the server's
 * value) and no server config ever applied. This locks the hex deviceId decoding.
 */
class ConfigurationDTOParseTest {

    @Test
    fun decodesConfigWithHexObjectIdDeviceId() {
        val json = """
            {
              "deviceId": "6a4c20aea7d8017f3540156d",
              "touchBalanceCheckUssd": "*220#",
              "alfaBalanceCheckUssd": "*11#",
              "touchThirdPartyRechargeTemplate": "*300*{phone}#{card}",
              "touchCreditTransferSmsTemplate": "{phone}T{amount}",
              "touchCreditTransferDestination": "1199",
              "alfaThirdPartyRechargeSmsTemplate": "{phone}R{code}",
              "alfaCreditTransferDestination": "1313",
              "alfaCreditTransferSmsTemplate": "{phone}T{amount}",
              "alfaThirdPartyRechargeDestination": "1313",
              "touchMinimumAllowedBalance": 0.5,
              "alfaMinimumAllowedBalance": 0.5,
              "touchSimBalance": 4.26,
              "touchSimValidityDate": "10-06-27",
              "touchCreditTransferMessageFee": 0.16,
              "alfaCreditTransferMessageFee": 0.14
            }
        """.trimIndent()

        // The whole decode used to throw NumberFormatException here (deviceId: Int).
        val dto = Gson().fromJson(json, ConfigurationDTO::class.java)

        assertEquals("6a4c20aea7d8017f3540156d", dto.deviceId)
        assertEquals("*11#", dto.alfaBalanceCheckUssd)
        assertEquals("1199", dto.touchCreditTransferDestination)
        // The server config actually reaches the store now — notably the real
        // min-balance reserve (0.5), not the hardcoded 20.0 fallback.
        assertEquals(0.5, dto.touchMinimumAllowedBalance, 0.0001)
        assertEquals(4.26, dto.touchSimBalance, 0.0001)
    }
}
