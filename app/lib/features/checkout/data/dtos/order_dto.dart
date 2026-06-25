import 'package:json_annotation/json_annotation.dart';

import '../../../../core/i18n/i18n_string.dart';
import '../../domain/entities/order.dart';

part 'order_dto.g.dart';

@JsonSerializable()
class RecipientDto {
  const RecipientDto({this.name = '', this.country = '', this.detail = ''});

  final String name;
  final String country;
  final String detail;

  factory RecipientDto.fromJson(Map<String, dynamic> json) =>
      _$RecipientDtoFromJson(json);

  Map<String, dynamic> toJson() => _$RecipientDtoToJson(this);

  OrderRecipient toEntity() =>
      OrderRecipient(name: name, country: country, detail: detail);
}

@JsonSerializable(explicitToJson: true)
class OrderItemDto {
  const OrderItemDto({
    this.productId = '',
    this.variantId = '',
    this.title,
    this.denomination = '',
    this.category = '',
    this.qty = 0,
    this.price = 0,
    this.fulfillmentType = 'code',
    this.playerId,
    this.recipient,
  });

  final String productId;
  final String variantId;
  final Map<String, dynamic>? title;
  final String denomination;
  final String category;
  final int qty;
  final double price;
  final String fulfillmentType;
  final String? playerId;
  final RecipientDto? recipient;

  factory OrderItemDto.fromJson(Map<String, dynamic> json) =>
      _$OrderItemDtoFromJson(json);

  Map<String, dynamic> toJson() => _$OrderItemDtoToJson(this);

  OrderItem toEntity() => OrderItem(
        productId: productId,
        variantId: variantId,
        title: I18nString.fromJson(title),
        denomination: denomination,
        category: category,
        qty: qty,
        price: price,
        fulfillmentType: fulfillmentType,
        playerId: playerId,
        recipient: recipient?.toEntity(),
      );
}

@JsonSerializable()
class TimelineEventDto {
  const TimelineEventDto({this.status = '', this.note = '', this.at});

  final String status;
  final String note;
  final String? at;

  factory TimelineEventDto.fromJson(Map<String, dynamic> json) =>
      _$TimelineEventDtoFromJson(json);

  Map<String, dynamic> toJson() => _$TimelineEventDtoToJson(this);

  TimelineEvent toEntity() => TimelineEvent(
        status: status,
        note: note,
        at: at == null ? null : DateTime.tryParse(at!),
      );
}

@JsonSerializable(explicitToJson: true)
class FulfillmentDto {
  const FulfillmentDto({
    this.deliveredCode,
    this.creditedToId,
    this.transferRef,
    this.statusTimeline,
  });

  final String? deliveredCode;
  final String? creditedToId;
  final String? transferRef;
  final List<TimelineEventDto>? statusTimeline;

  factory FulfillmentDto.fromJson(Map<String, dynamic> json) =>
      _$FulfillmentDtoFromJson(json);

  Map<String, dynamic> toJson() => _$FulfillmentDtoToJson(this);

  Fulfillment toEntity() => Fulfillment(
        deliveredCode: deliveredCode,
        creditedToId: creditedToId,
        transferRef: transferRef,
        statusTimeline:
            (statusTimeline ?? const []).map((e) => e.toEntity()).toList(),
      );
}

@JsonSerializable(explicitToJson: true)
class OrderDto {
  const OrderDto({
    required this.id,
    this.userId = '',
    this.items,
    this.subtotal = 0,
    this.total = 0,
    this.currency = 'USD',
    this.paymentMethod = '',
    this.status = '',
    this.fulfillment,
    this.createdAt,
    this.updatedAt,
  });

  final String id;
  final String userId;
  final List<OrderItemDto>? items;
  final double subtotal;
  final double total;
  final String currency;
  final String paymentMethod;
  final String status;
  final FulfillmentDto? fulfillment;
  final String? createdAt;
  final String? updatedAt;

  factory OrderDto.fromJson(Map<String, dynamic> json) =>
      _$OrderDtoFromJson(json);

  Map<String, dynamic> toJson() => _$OrderDtoToJson(this);

  Order toEntity() => Order(
        id: id,
        userId: userId,
        items: (items ?? const []).map((e) => e.toEntity()).toList(),
        subtotal: subtotal,
        total: total,
        currency: currency,
        paymentMethod: paymentMethod,
        status: orderStatusFromString(status),
        fulfillment: fulfillment?.toEntity() ?? const Fulfillment(),
        createdAt: createdAt == null ? null : DateTime.tryParse(createdAt!),
        updatedAt: updatedAt == null ? null : DateTime.tryParse(updatedAt!),
      );
}
