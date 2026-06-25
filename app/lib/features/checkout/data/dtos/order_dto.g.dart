// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'order_dto.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

RecipientDto _$RecipientDtoFromJson(Map<String, dynamic> json) => RecipientDto(
  name: json['name'] as String? ?? '',
  country: json['country'] as String? ?? '',
  detail: json['detail'] as String? ?? '',
);

Map<String, dynamic> _$RecipientDtoToJson(RecipientDto instance) =>
    <String, dynamic>{
      'name': instance.name,
      'country': instance.country,
      'detail': instance.detail,
    };

OrderItemDto _$OrderItemDtoFromJson(Map<String, dynamic> json) => OrderItemDto(
  productId: json['productId'] as String? ?? '',
  variantId: json['variantId'] as String? ?? '',
  title: json['title'] as Map<String, dynamic>?,
  denomination: json['denomination'] as String? ?? '',
  category: json['category'] as String? ?? '',
  qty: (json['qty'] as num?)?.toInt() ?? 0,
  price: (json['price'] as num?)?.toDouble() ?? 0,
  fulfillmentType: json['fulfillmentType'] as String? ?? 'code',
  playerId: json['playerId'] as String?,
  recipient: json['recipient'] == null
      ? null
      : RecipientDto.fromJson(json['recipient'] as Map<String, dynamic>),
);

Map<String, dynamic> _$OrderItemDtoToJson(OrderItemDto instance) =>
    <String, dynamic>{
      'productId': instance.productId,
      'variantId': instance.variantId,
      'title': instance.title,
      'denomination': instance.denomination,
      'category': instance.category,
      'qty': instance.qty,
      'price': instance.price,
      'fulfillmentType': instance.fulfillmentType,
      'playerId': instance.playerId,
      'recipient': instance.recipient?.toJson(),
    };

TimelineEventDto _$TimelineEventDtoFromJson(Map<String, dynamic> json) =>
    TimelineEventDto(
      status: json['status'] as String? ?? '',
      note: json['note'] as String? ?? '',
      at: json['at'] as String?,
    );

Map<String, dynamic> _$TimelineEventDtoToJson(TimelineEventDto instance) =>
    <String, dynamic>{
      'status': instance.status,
      'note': instance.note,
      'at': instance.at,
    };

FulfillmentDto _$FulfillmentDtoFromJson(Map<String, dynamic> json) =>
    FulfillmentDto(
      deliveredCode: json['deliveredCode'] as String?,
      creditedToId: json['creditedToId'] as String?,
      transferRef: json['transferRef'] as String?,
      statusTimeline: (json['statusTimeline'] as List<dynamic>?)
          ?.map((e) => TimelineEventDto.fromJson(e as Map<String, dynamic>))
          .toList(),
    );

Map<String, dynamic> _$FulfillmentDtoToJson(
  FulfillmentDto instance,
) => <String, dynamic>{
  'deliveredCode': instance.deliveredCode,
  'creditedToId': instance.creditedToId,
  'transferRef': instance.transferRef,
  'statusTimeline': instance.statusTimeline?.map((e) => e.toJson()).toList(),
};

OrderDto _$OrderDtoFromJson(Map<String, dynamic> json) => OrderDto(
  id: json['id'] as String,
  userId: json['userId'] as String? ?? '',
  items: (json['items'] as List<dynamic>?)
      ?.map((e) => OrderItemDto.fromJson(e as Map<String, dynamic>))
      .toList(),
  subtotal: (json['subtotal'] as num?)?.toDouble() ?? 0,
  total: (json['total'] as num?)?.toDouble() ?? 0,
  currency: json['currency'] as String? ?? 'USD',
  paymentMethod: json['paymentMethod'] as String? ?? '',
  status: json['status'] as String? ?? '',
  fulfillment: json['fulfillment'] == null
      ? null
      : FulfillmentDto.fromJson(json['fulfillment'] as Map<String, dynamic>),
  createdAt: json['createdAt'] as String?,
  updatedAt: json['updatedAt'] as String?,
);

Map<String, dynamic> _$OrderDtoToJson(OrderDto instance) => <String, dynamic>{
  'id': instance.id,
  'userId': instance.userId,
  'items': instance.items?.map((e) => e.toJson()).toList(),
  'subtotal': instance.subtotal,
  'total': instance.total,
  'currency': instance.currency,
  'paymentMethod': instance.paymentMethod,
  'status': instance.status,
  'fulfillment': instance.fulfillment?.toJson(),
  'createdAt': instance.createdAt,
  'updatedAt': instance.updatedAt,
};
