package com.example.mobilebridgev2.dto
object CommandResultCodes {

    /*
     * 1000–1099: General SMS commands
     */

    const val SMS_SUCCESS = 1000

    const val SMS_NO_RECIPIENT = 1001
    const val SMS_NO_MESSAGE = 1002

    const val SMS_SEND_FAILED = 1010


    /*
     * 2000–2099: Credit transfer commands
     */

    const val CREDIT_TRANSFER_SUCCESS = 2000
    const val CREDIT_TRANSFER_PARTIALLY_COMPLETED = 2001

    const val CREDIT_TRANSFER_NO_RECIPIENT = 2010
    const val CREDIT_TRANSFER_NO_AMOUNT = 2011
    const val CREDIT_TRANSFER_INVALID_AMOUNT = 2012
    const val CREDIT_TRANSFER_MISSING_FIELDS = 2013

    const val CREDIT_TRANSFER_INSUFFICIENT_BALANCE = 2020
    const val CREDIT_TRANSFER_SMS_SEND_FAILED = 2021
    const val CREDIT_TRANSFER_REPLY_TIMEOUT = 2022
    const val CREDIT_TRANSFER_PROVIDER_REJECTED = 2023
    const val CREDIT_TRANSFER_REPLY_PARSE_FAILED = 2024


    /*
     * 3000–3099: Touch third-party recharge commands
     */

    const val TOUCH_RECHARGE_SUCCESS = 3000

    const val TOUCH_RECHARGE_NO_RECIPIENT = 3010
    const val TOUCH_RECHARGE_NO_CARD_CODE = 3011
    const val TOUCH_RECHARGE_INVALID_CARD_CODE = 3012

    const val TOUCH_RECHARGE_USSD_SEND_FAILED = 3020
    const val TOUCH_RECHARGE_REPLY_TIMEOUT = 3021
    const val TOUCH_RECHARGE_PROVIDER_REJECTED = 3022
    const val TOUCH_RECHARGE_REPLY_PARSE_FAILED = 3023


    /*
     * 4000–4099: Alfa third-party recharge commands
     */

    const val ALFA_RECHARGE_SUCCESS = 4000

    const val ALFA_RECHARGE_NO_RECIPIENT = 4010
    const val ALFA_RECHARGE_NO_CARD_CODE = 4011
    const val ALFA_RECHARGE_INVALID_CARD_CODE = 4012

    const val ALFA_RECHARGE_SMS_SEND_FAILED = 4020
    const val ALFA_RECHARGE_REPLY_TIMEOUT = 4021
    const val ALFA_RECHARGE_PROVIDER_REJECTED = 4022
    const val ALFA_RECHARGE_REPLY_PARSE_FAILED = 4023


    /*
     * 5000–5099: USSD and balance-check commands
     */

    const val BALANCE_CHECK_SUCCESS = 5000

    const val USSD_NO_PROVIDER = 5010
    const val USSD_NO_SUBSCRIPTION_ID = 5011
    const val USSD_CODE_MISSING = 5012

    const val USSD_SEND_FAILED = 5020
    const val USSD_REPLY_TIMEOUT = 5021
    const val USSD_REPLY_PARSE_FAILED = 5022


    /*
     * 9000–9099: General command and provider errors
     */

    const val PROVIDER_NOT_SUPPORTED = 9000
    const val COMMAND_NOT_SUPPORTED = 9001
    const val COMMAND_EXECUTION_FAILED = 9002
    const val COMMAND_RESULT_REPORT_FAILED = 9003
}
