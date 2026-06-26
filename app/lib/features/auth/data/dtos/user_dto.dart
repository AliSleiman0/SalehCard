import 'package:json_annotation/json_annotation.dart';

import '../../domain/entities/user.dart';

part 'user_dto.g.dart';

@JsonSerializable()
class UserDto {
  const UserDto({
    required this.id,
    required this.email,
    required this.role,
    required this.locale,
    this.phone,
    this.walletBalance = 0,
    this.loyaltyPoints = 0,
    this.savedPlayerIds = const [],
  });

  final String id;
  final String email;
  final String? phone;
  final String role;
  final String locale;
  final double walletBalance;
  final int loyaltyPoints;

  final List<String> savedPlayerIds;

  factory UserDto.fromJson(Map<String, dynamic> json) =>
      _$UserDtoFromJson(json);

  Map<String, dynamic> toJson() => _$UserDtoToJson(this);

  User toEntity() => User(
        id: id,
        email: email,
        phone: phone,
        role: role,
        locale: locale,
        walletBalance: walletBalance,
        loyaltyPoints: loyaltyPoints,
        savedPlayerIds: savedPlayerIds,
      );
}
