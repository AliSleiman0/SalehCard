package com.example.mobilebridgev2.websocket

import com.example.mobilebridgev2.dto.ConfigurationDTO

data class ReceivedMessageDTO(
    val topupObject: TopupDTO?,
    val configurationObject: ConfigurationDTO?
)
