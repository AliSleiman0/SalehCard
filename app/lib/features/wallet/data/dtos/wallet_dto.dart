import 'package:json_annotation/json_annotation.dart';

import '../../domain/entities/wallet.dart';

part 'wallet_dto.g.dart';

@JsonSerializable()
class WalletTxDto {
  const WalletTxDto({
    required this.id,
    this.userId = '',
    this.type = '',
    this.amount = 0,
    this.balanceAfter = 0,
    this.method = '',
    this.ref = '',
    this.createdAt,
  });

  final String id;
  final String userId;
  final String type;
  final double amount;
  final double balanceAfter;
  final String method;
  final String ref;
  final String? createdAt;

  factory WalletTxDto.fromJson(Map<String, dynamic> json) =>
      _$WalletTxDtoFromJson(json);

  Map<String, dynamic> toJson() => _$WalletTxDtoToJson(this);

  WalletTx toEntity() => WalletTx(
        id: id,
        type: walletTxTypeFromString(type),
        amount: amount,
        balanceAfter: balanceAfter,
        method: method,
        ref: ref,
        createdAt: createdAt == null ? null : DateTime.tryParse(createdAt!),
      );
}

@JsonSerializable(explicitToJson: true)
class WalletDto {
  const WalletDto({this.balance = 0, this.transactions});

  final double balance;
  final List<WalletTxDto>? transactions;

  factory WalletDto.fromJson(Map<String, dynamic> json) =>
      _$WalletDtoFromJson(json);

  Map<String, dynamic> toJson() => _$WalletDtoToJson(this);

  Wallet toEntity() => Wallet(
        balance: balance,
        transactions:
            (transactions ?? const []).map((e) => e.toEntity()).toList(),
      );
}
