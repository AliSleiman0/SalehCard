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
    this.walletBalance = 0,
    this.loyaltyPoints = 0,
  });

  final String id;
  final String email;
  final String role;
  final String locale;
  final double walletBalance;
  final int loyaltyPoints;

  factory UserDto.fromJson(Map<String, dynamic> json) =>
      _$UserDtoFromJson(json);

  Map<String, dynamic> toJson() => _$UserDtoToJson(this);

  User toEntity() => User(
        id: id,
        email: email,
        role: role,
        locale: locale,
        walletBalance: walletBalance,
        loyaltyPoints: loyaltyPoints,
      );
}
