package com.example.mobilebridgev2.dto

data class CommandResultDTO(
    val id: String,
    val recipientNumber: String?,
    val type: String,
    val timestamp: Long = System.currentTimeMillis(),
    val statusCode: Int,
    val amount: Double?,
    val cardCode: String?,
    val billingAmount: Double?,
    val provider: String,
    val balance: Double?,
    val validityDate: String?,
    val errorMessage: String?
)