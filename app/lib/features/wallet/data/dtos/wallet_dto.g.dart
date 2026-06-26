// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'wallet_dto.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

WalletTxDto _$WalletTxDtoFromJson(Map<String, dynamic> json) => WalletTxDto(
  id: json['id'] as String,
  userId: json['userId'] as String? ?? '',
  type: json['type'] as String? ?? '',
  amount: (json['amount'] as num?)?.toDouble() ?? 0,
  balanceAfter: (json['balanceAfter'] as num?)?.toDouble() ?? 0,
  method: json['method'] as String? ?? '',
  ref: json['ref'] as String? ?? '',
  createdAt: json['createdAt'] as String?,
);

Map<String, dynamic> _$WalletTxDtoToJson(WalletTxDto instance) =>
    <String, dynamic>{
      'id': instance.id,
      'userId': instance.userId,
      'type': instance.type,
      'amount': instance.amount,
      'balanceAfter': instance.balanceAfter,
      'method': instance.method,
      'ref': instance.ref,
      'createdAt': instance.createdAt,
    };

WalletDto _$WalletDtoFromJson(Map<String, dynamic> json) => WalletDto(
  balance: (json['balance'] as num?)?.toDouble() ?? 0,
  transactions: (json['transactions'] as List<dynamic>?)
      ?.map((e) => WalletTxDto.fromJson(e as Map<String, dynamic>))
      .toList(),
);

Map<String, dynamic> _$WalletDtoToJson(WalletDto instance) => <String, dynamic>{
  'balance': instance.balance,
  'transactions': instance.transactions?.map((e) => e.toJson()).toList(),
};
