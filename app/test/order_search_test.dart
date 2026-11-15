import 'package:flutter_test/flutter_test.dart';
import 'package:salehcard_app/core/format/order_number.dart';
import 'package:salehcard_app/core/i18n/i18n_string.dart';
import 'package:salehcard_app/features/checkout/domain/entities/order.dart';
import 'package:salehcard_app/features/orders/domain/order_search.dart';

OrderItem _item({
  String? playerId,
  OrderRecipient? recipient,
  List<OrderLineField> fields = const [],
}) => OrderItem(
  productId: 'p1',
  variantId: 'v1',
  title: const I18nString({
    'en': 'PUBG Mobile 60 UC',
    'ar': 'ببجي موبايل ٦٠ شدة',
  }),
  denomination: '60 UC',
  category: 'game',
  qty: 1,
  price: 1,
  fulfillmentType: 'account_credit',
  playerId: playerId,
  recipient: recipient,
  fields: fields,
);

Order _order([List<OrderItem>? items]) => Order(
  id: '6731f0c2ab45de67a3f9c1d2',
  userId: 'u1',
  items: items ?? [_item()],
  subtotal: 1,
  total: 1,
  currency: 'USD',
  paymentMethod: 'wallet',
  status: OrderStatus.completed,
  fulfillment: const Fulfillment(),
);

OrderLineField _field(String key, String value) =>
    OrderLineField(key: key, label: I18nString({'en': key}), value: value);

void main() {
  final order = _order();

  test('normalizeOrderQuery trims, lower-cases and drops a leading #', () {
    expect(normalizeOrderQuery('  #A3F9 '), 'a3f9');
    expect(normalizeOrderQuery('##x'), 'x');
    expect(normalizeOrderQuery('   '), '');
  });

  test('an empty query matches everything', () {
    expect(orderMatchesQuery(order, ''), isTrue);
    expect(orderMatchesQuery(order, ' # '), isTrue);
  });

  test('matches the order number shown on the row, with or without #', () {
    expect(orderNumber(order.id), '#A3F9C1D2');
    expect(orderMatchesQuery(order, '#A3F9C1D2'), isTrue);
    expect(orderMatchesQuery(order, 'a3f9c1d2'), isTrue);
    expect(orderMatchesQuery(order, '#a3f9'), isTrue);
  });

  test('matches the full order id and a prefix of it', () {
    expect(orderMatchesQuery(order, '6731f0c2ab45de67a3f9c1d2'), isTrue);
    expect(orderMatchesQuery(order, '6731F0'), isTrue);
  });

  test('does not match a run from the middle of the id', () {
    expect(orderMatchesQuery(order, '45de'), isFalse);
  });

  test('matches a player ID and a checkout field value', () {
    final o = _order([
      _item(
        playerId: '5123456789',
        fields: [_field('id', '5123456789'), _field('zone', 'Zone-2211')],
      ),
    ]);
    expect(orderMatchesQuery(o, '512345'), isTrue);
    expect(orderMatchesQuery(o, 'zone-2211'), isTrue);
  });

  test('matches the transfer recipient name and detail', () {
    final o = _order([
      _item(
        recipient: const OrderRecipient(
          name: 'Ali Hassan',
          country: 'LB',
          detail: '+961 70 123 456',
        ),
      ),
    ]);
    expect(orderMatchesQuery(o, 'hassan'), isTrue);
    expect(orderMatchesQuery(o, '70 123'), isTrue);
  });

  test('matches the product title in English and Arabic', () {
    expect(orderMatchesQuery(order, 'pubg'), isTrue);
    expect(orderMatchesQuery(order, 'ببجي'), isTrue);
  });

  test('returns false when nothing matches', () {
    expect(orderMatchesQuery(order, 'free fire'), isFalse);
  });

  test('never matches inside a photo (QR / barcode) URL', () {
    final o = _order([
      _item(
        fields: [
          _field(
            'qr',
            'https://salehcardstorage.blob.core.windows.net/product-images/order-files/9f8e7d.jpg',
          ),
        ],
      ),
    ]);
    expect(orderMatchesQuery(o, 'blob'), isFalse);
    expect(orderMatchesQuery(o, '9f8e7d'), isFalse);
    expect(orderMatchesQuery(o, 'https'), isFalse);
  });
}
