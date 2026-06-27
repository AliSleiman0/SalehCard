// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'user_dto.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

UserDto _$UserDtoFromJson(Map<String, dynamic> json) => UserDto(
  id: json['id'] as String,
  email: json['email'] as String,
  role: json['role'] as String,
  locale: json['locale'] as String,
  name: json['name'] as String? ?? '',
  phone: json['phone'] as String?,
  walletBalance: (json['walletBalance'] as num?)?.toDouble() ?? 0,
  loyaltyPoints: (json['loyaltyPoints'] as num?)?.toInt() ?? 0,
  savedPlayerIds:
      (json['savedPlayerIds'] as List<dynamic>?)
          ?.map((e) => e as String)
          .toList() ??
      const [],
);

Map<String, dynamic> _$UserDtoToJson(UserDto instance) => <String, dynamic>{
  'id': instance.id,
  'name': instance.name,
  'email': instance.email,
  'phone': instance.phone,
  'role': instance.role,
  'locale': instance.locale,
  'walletBalance': instance.walletBalance,
  'loyaltyPoints': instance.loyaltyPoints,
  'savedPlayerIds': instance.savedPlayerIds,
};
