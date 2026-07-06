package com.example.mobilebridgev2

import android.Manifest
import android.content.Context
import android.os.Build
import androidx.annotation.RequiresApi
import androidx.annotation.RequiresPermission
import com.example.mobilebridgev2.dto.CommandDTO
import com.example.mobilebridgev2.dto.CommandResultCodes
import com.example.mobilebridgev2.dto.CommandResultDTO

object CommandDispatcher {
    @RequiresApi(Build.VERSION_CODES.S)
    @RequiresPermission(Manifest.permission.CALL_PHONE)
    suspend fun dispatch(command: CommandDTO, context: Context): CommandResultDTO{
        return when(command.provider.lowercase()){
            "alfa" -> when(command.commandType){
                "SEND_SMS" -> CommandExecutor.smsTransfer(command, context)
                "TRANSFER_CREDIT" -> CommandExecutor.transferCredits(command, context)
                "RECHARGE_LINE" -> CommandExecutor.rechargeAlfa(command, context)
                "CHECK-BALANCE" -> CommandExecutor.checkBalance(command, context)
                else -> CommandResultDTO(
                    id = command.commandId,
                    recipientNumber = command.recipientNumber,
                    type = command.commandType,
                    timestamp = System.currentTimeMillis(),
                    statusCode = CommandResultCodes.COMMAND_NOT_SUPPORTED,
                    amount = null,
                    cardCode = null,
                    billingAmount = null,
                    provider = command.provider,
                    balance = null,
                    validityDate = null,
                    errorMessage = "Command not available"
                )
            }
            "touch" -> when(command.commandType){
                "SEND_SMS" -> CommandExecutor.smsTransfer(command, context)
                "TRANSFER_CREDIT" -> CommandExecutor.transferCredits(command, context)
                "RECHARGE_LINE" -> CommandExecutor.rechargeTouch(command, context)
                "CHECK-BALANCE" -> CommandExecutor.checkBalance(command, context)
                else -> CommandResultDTO(
                    id = command.commandId,
                    recipientNumber = command.recipientNumber,
                    type = command.commandType,
                    timestamp = System.currentTimeMillis(),
                    statusCode = CommandResultCodes.COMMAND_NOT_SUPPORTED,
                    amount = null,
                    cardCode = null,
                    billingAmount = null,
                    provider = command.provider,
                    balance = null,
                    validityDate = null,
                    errorMessage = "Command not available"
                )
            }
            else -> CommandResultDTO(
                id = command.commandId,
                recipientNumber = command.recipientNumber,
                type = command.commandType,
                timestamp = System.currentTimeMillis(),
                statusCode = CommandResultCodes.PROVIDER_NOT_SUPPORTED,
                amount = null,
                cardCode = null,
                billingAmount = null,
                provider = command.provider,
                balance = null,
                validityDate = null,
                errorMessage = "Provider not available"
            )
        }
    }
}