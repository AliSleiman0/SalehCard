package com.example.mobilebridgev2.dto

object LogErrorCodes {

    /*
     * 1000–1099: Foreground service and bridge-engine errors
     */

    const val SERVICE_START_FAILED = 1000
    const val BRIDGE_ENGINE_START_FAILED = 1001
    const val FOREGROUND_NOTIFICATION_FAILED = 1002
    const val SERVICE_ALREADY_RUNNING = 1003


    /*
     * 2000–2099: Permission errors
     */

    const val REQUIRED_PERMISSIONS_NOT_GRANTED = 2000
    const val SEND_SMS_PERMISSION_NOT_GRANTED = 2001
    const val RECEIVE_SMS_PERMISSION_NOT_GRANTED = 2002
    const val READ_SMS_PERMISSION_NOT_GRANTED = 2003
    const val READ_PHONE_STATE_PERMISSION_NOT_GRANTED = 2004
    const val CALL_PHONE_PERMISSION_NOT_GRANTED = 2005
    const val NOTIFICATION_PERMISSION_NOT_GRANTED = 2006


    /*
     * 3000–3099: Configuration and device-state errors
     */

    const val CONFIGURATION_FETCH_FAILED = 3000
    const val CONFIGURATION_INVALID = 3001
    const val DEVICE_ID_MISSING = 3002
    const val AUTH_TOKEN_MISSING = 3003
    const val PROVIDER_CONFIGURATION_MISSING = 3004


    /*
     * 4000–4099: SIM and provider-detection errors
     */

    const val ACTIVE_SIM_READ_FAILED = 4000
    const val NO_ACTIVE_SIM_FOUND = 4001
    const val TOUCH_SIM_NOT_FOUND = 4002
    const val ALFA_SIM_NOT_FOUND = 4003
    const val INVALID_PROVIDER = 4004
    const val SUBSCRIPTION_ID_NOT_AVAILABLE = 4005


    /*
     * 5000–5099: Polling and command-queue errors
     */

    const val POLLING_LOOP_START_FAILED = 5000
    const val POLLING_REQUEST_FAILED = 5001
    const val COMMAND_QUEUE_WORKER_START_FAILED = 5002
    const val COMMAND_QUEUE_INSERT_FAILED = 5003
    const val COMMAND_DISPATCH_FAILED = 5004
    const val COMMAND_EXECUTION_FAILED = 5005


    /*
     * 6000–6099: SMS and USSD infrastructure errors
     */

    const val SMS_MANAGER_INITIALIZATION_FAILED = 6000
    const val SMS_SEND_OPERATION_FAILED = 6001
    const val SMS_REPLY_RECEIVER_FAILED = 6002
    const val SMS_REPLY_WAIT_FAILED = 6003
    const val USSD_OPERATION_FAILED = 6004
    const val USSD_REPLY_PARSE_FAILED = 6005


    /*
     * 7000–7099: Balance-monitor errors
     */

    const val BALANCE_MONITOR_START_FAILED = 7000
    const val TOUCH_BALANCE_CHECK_FAILED = 7001
    const val ALFA_BALANCE_CHECK_FAILED = 7002
    const val BALANCE_STATE_UPDATE_FAILED = 7003


    /*
     * 8000–8099: WebSocket and backend-reporting errors
     */

    const val WEBSOCKET_CONNECTION_FAILED = 8000
    const val WEBSOCKET_DISCONNECTED = 8001
    const val WEBSOCKET_MESSAGE_PARSE_FAILED = 8002
    const val WEBSOCKET_MESSAGE_SEND_FAILED = 8003
    const val COMMAND_RESULT_REPORT_FAILED = 8004
    const val LOG_REPORT_FAILED = 8005
}