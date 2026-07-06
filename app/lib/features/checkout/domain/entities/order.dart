import '../../../../core/i18n/i18n_string.dart';
import '../../../payments/domain/entities/payment_intent.dart';

/// Lifecycle status of an order (mirrors the backend `OrderStatus`).
enum OrderStatus { pending, processing, completed, failed, refunded, unknown }

OrderStatus orderStatusFromString(String value) {
  switch (value) {
    case 'pending':
      return OrderStatus.pending;
    case 'processing':
      return OrderStatus.processing;
    case 'completed':
      return OrderStatus.completed;
    case 'failed':
      return OrderStatus.failed;
    case 'refunded':
      return OrderStatus.refunded;
    default:
      return OrderStatus.unknown;
  }
}

/// Destination for a manual money-transfer fulfillment.
class OrderRecipient {
  const OrderRecipient({
    required this.name,
    required this.country,
    required this.detail,
  });

  final String name;
  final String country;
  final String detail;
}

/// A status transition recorded on a fulfillment.
class TimelineEvent {
  const TimelineEvent({required this.status, required this.note, this.at});

  final String status;
  final String note;
  final DateTime? at;
}

/// Delivery details for an order.
class Fulfillment {
  const Fulfillment({
    this.deliveredCode,
    this.creditedToId,
    this.transferRef,
    this.statusTimeline = const [],
  });

  final String? deliveredCode;
  final String? creditedToId;
  final String? transferRef;
  final List<TimelineEvent> statusTimeline;
}

/// A single purchased line. Title/denomination/category are purchase-time
/// snapshots so order history renders even if the catalog later changes.
class OrderItem {
  const OrderItem({
    required this.productId,
    required this.variantId,
    required this.title,
    required this.denomination,
    required this.category,
    required this.qty,
    required this.price,
    required this.fulfillmentType,
    this.playerId,
    this.recipient,
  });

  final String productId;
  final String variantId;
  final I18nString title;
  final String denomination;
  final String category;
  final int qty;
  final double price;
  final String fulfillmentType;
  final String? playerId;
  final OrderRecipient? recipient;

  double get lineTotal => price * qty;
}

/// Root aggregate for a customer purchase.
class Order {
  const Order({
    required this.id,
    required this.userId,
    required this.items,
    required this.subtotal,
    required this.total,
    required this.currency,
    required this.paymentMethod,
    required this.status,
    required this.fulfillment,
    this.createdAt,
    this.updatedAt,
    this.paymentIntent,
  });

  final String id;
  final String userId;
  final List<OrderItem> items;
  final double subtotal;
  final double total;
  final String currency;
  final String paymentMethod;
  final OrderStatus status;
  final Fulfillment fulfillment;
  final DateTime? createdAt;
  final DateTime? updatedAt;

  /// The on-chain deposit intent, present only on the response to placing a
  /// USDT order (the client shows its waiting-for-payment screen). Null for
  /// wallet/card orders and for orders re-fetched via GET /orders.
  final PaymentIntent? paymentIntent;

  /// Returns a copy with [paymentIntent] attached (used by the order datasource
  /// to carry the intent parsed from the place-order response).
  Order withPaymentIntent(PaymentIntent? intent) => Order(
        id: id,
        userId: userId,
        items: items,
        subtotal: subtotal,
        total: total,
        currency: currency,
        paymentMethod: paymentMethod,
        status: status,
        fulfillment: fulfillment,
        createdAt: createdAt,
        updatedAt: updatedAt,
        paymentIntent: intent,
      );

  bool get isProcessing => status == OrderStatus.processing;
  bool get isCompleted => status == OrderStatus.completed;

  /// True when the order delivered a redeemable code.
  bool get hasDeliveredCode =>
      fulfillment.deliveredCode != null &&
      fulfillment.deliveredCode!.isNotEmpty;

  /// Short, human-friendly order reference (last 6 chars of the id, upper-cased).
  String get reference {
    if (id.length <= 6) return id.toUpperCase();
    return id.substring(id.length - 6).toUpperCase();
  }
}

/// One labeled customer input on an order line (e.g. Account ID, Zone ID). Only
/// key + value are sent; the server resolves the display label from the product.
class PlaceOrderField {
  const PlaceOrderField({required this.key, required this.value});

  final String key;
  final String value;
}

/// A single requested order line (client sends only what it may choose — never
/// the price or fulfillment type; the server derives those).
class PlaceOrderLine {
  const PlaceOrderLine({
    required this.productId,
    required this.variantId,
    required this.qty,
    this.playerId,
    this.recipient,
    this.fields = const [],
  });

  final String productId;
  final String variantId;
  final int qty;
  final String? playerId;
  final OrderRecipient? recipient;

  /// Structured per-field inputs for account_credit lines (Account ID, Zone ID,
  /// Email, …). Empty for code/transfer lines.
  final List<PlaceOrderField> fields;
}

/// Payload for placing an order.
class PlaceOrderInput {
  const PlaceOrderInput({
    required this.items,
    required this.paymentMethod,
    this.currency = 'USD',
    this.promoCode,
  });

  final List<PlaceOrderLine> items;
  final String paymentMethod; // wallet | card | usdt
  final String currency;
  final String? promoCode;
}
