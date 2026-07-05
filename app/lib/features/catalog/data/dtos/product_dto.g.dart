// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'product_dto.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

VariantDto _$VariantDtoFromJson(Map<String, dynamic> json) => VariantDto(
  id: json['id'] as String,
  denomination: json['denomination'] as String? ?? '',
  price: (json['price'] as num?)?.toDouble() ?? 0,
  offerPrice: (json['offerPrice'] as num?)?.toDouble(),
);

Map<String, dynamic> _$VariantDtoToJson(VariantDto instance) =>
    <String, dynamic>{
      'id': instance.id,
      'denomination': instance.denomination,
      'price': instance.price,
      'offerPrice': instance.offerPrice,
    };

ProductOfferDto _$ProductOfferDtoFromJson(Map<String, dynamic> json) =>
    ProductOfferDto(
      discountType: json['discountType'] as String? ?? 'percent',
      discountValue: (json['discountValue'] as num?)?.toDouble() ?? 0,
      originalFromPrice: (json['originalFromPrice'] as num?)?.toDouble() ?? 0,
      offerFromPrice: (json['offerFromPrice'] as num?)?.toDouble() ?? 0,
      endsAt: json['endsAt'] == null
          ? null
          : DateTime.parse(json['endsAt'] as String),
    );

Map<String, dynamic> _$ProductOfferDtoToJson(ProductOfferDto instance) =>
    <String, dynamic>{
      'discountType': instance.discountType,
      'discountValue': instance.discountValue,
      'originalFromPrice': instance.originalFromPrice,
      'offerFromPrice': instance.offerFromPrice,
      'endsAt': instance.endsAt?.toIso8601String(),
    };

RatingsDto _$RatingsDtoFromJson(Map<String, dynamic> json) => RatingsDto(
  average: (json['average'] as num?)?.toDouble() ?? 0,
  count: (json['count'] as num?)?.toInt() ?? 0,
);

Map<String, dynamic> _$RatingsDtoToJson(RatingsDto instance) =>
    <String, dynamic>{'average': instance.average, 'count': instance.count};

InputFieldConstraintsDto _$InputFieldConstraintsDtoFromJson(
  Map<String, dynamic> json,
) => InputFieldConstraintsDto(
  min: (json['min'] as num?)?.toDouble(),
  max: (json['max'] as num?)?.toDouble(),
  options: (json['options'] as List<dynamic>?)
      ?.map((e) => e as String)
      .toList(),
);

Map<String, dynamic> _$InputFieldConstraintsDtoToJson(
  InputFieldConstraintsDto instance,
) => <String, dynamic>{
  'min': instance.min,
  'max': instance.max,
  'options': instance.options,
};

InputFieldDto _$InputFieldDtoFromJson(Map<String, dynamic> json) =>
    InputFieldDto(
      key: json['key'] as String,
      label: json['label'] as Map<String, dynamic>?,
      type: json['type'] as String? ?? 'text',
      constraints: json['constraints'] == null
          ? null
          : InputFieldConstraintsDto.fromJson(
              json['constraints'] as Map<String, dynamic>,
            ),
      sensitive: json['sensitive'] as bool? ?? false,
    );

Map<String, dynamic> _$InputFieldDtoToJson(InputFieldDto instance) =>
    <String, dynamic>{
      'key': instance.key,
      'label': instance.label,
      'type': instance.type,
      'constraints': instance.constraints?.toJson(),
      'sensitive': instance.sensitive,
    };

VerificationDto _$VerificationDtoFromJson(Map<String, dynamic> json) =>
    VerificationDto(
      provider: (json['provider'] as num?)?.toInt() ?? 0,
      app: json['app'] as String? ?? '',
    );

Map<String, dynamic> _$VerificationDtoToJson(VerificationDto instance) =>
    <String, dynamic>{'provider': instance.provider, 'app': instance.app};

ProductDto _$ProductDtoFromJson(Map<String, dynamic> json) => ProductDto(
  id: json['id'] as String,
  title: json['title'] as Map<String, dynamic>?,
  category: json['category'] as String?,
  images: (json['images'] as List<dynamic>?)?.map((e) => e as String).toList(),
  variants: (json['variants'] as List<dynamic>?)
      ?.map((e) => VariantDto.fromJson(e as Map<String, dynamic>))
      .toList(),
  stock: (json['stock'] as num?)?.toInt(),
  available: json['available'] as bool?,
  fulfillmentType: json['fulfillmentType'] as String?,
  inputFields: (json['inputFields'] as List<dynamic>?)
      ?.map((e) => InputFieldDto.fromJson(e as Map<String, dynamic>))
      .toList(),
  verification: json['verification'] == null
      ? null
      : VerificationDto.fromJson(json['verification'] as Map<String, dynamic>),
  ratings: json['ratings'] == null
      ? null
      : RatingsDto.fromJson(json['ratings'] as Map<String, dynamic>),
  offer: json['offer'] == null
      ? null
      : ProductOfferDto.fromJson(json['offer'] as Map<String, dynamic>),
);

Map<String, dynamic> _$ProductDtoToJson(ProductDto instance) =>
    <String, dynamic>{
      'id': instance.id,
      'title': instance.title,
      'category': instance.category,
      'images': instance.images,
      'variants': instance.variants?.map((e) => e.toJson()).toList(),
      'stock': instance.stock,
      'available': instance.available,
      'fulfillmentType': instance.fulfillmentType,
      'inputFields': instance.inputFields?.map((e) => e.toJson()).toList(),
      'verification': instance.verification?.toJson(),
      'ratings': instance.ratings?.toJson(),
      'offer': instance.offer?.toJson(),
    };
