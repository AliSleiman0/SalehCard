import 'package:flutter_test/flutter_test.dart';
import 'package:salehcard_app/core/format/money.dart';
import 'package:salehcard_app/core/i18n/i18n_string.dart';
import 'package:salehcard_app/features/cart/domain/entities/cart_item.dart';
import 'package:salehcard_app/features/catalog/data/dtos/product_dto.dart';
import 'package:salehcard_app/features/catalog/domain/entities/product.dart';
import 'package:salehcard_app/features/checkout/data/datasources/order_remote_data_source.dart';
import 'package:salehcard_app/features/checkout/data/dtos/order_dto.dart';
import 'package:salehcard_app/features/checkout/domain/entities/order.dart';

Map<String, dynamic> _coinsJson({
  Map<String, dynamic>? unitPricing,
  List<Map<String, dynamic>>? variants,
  List<Map<String, dynamic>>? inputFields,
}) =>
    <String, dynamic>{
      'id': 'p-uplive',
      'title': {'en': 'Up Live Coins'},
      'fulfillmentType': 'account_credit',
      'available': true,
      'variants': variants ??
          [
            {'id': 'v1', 'denomination': '1000 coins', 'price': 0.45},
          ],
      'inputFields': inputFields ??
          [
            {
              'key': 'playerId',
              'label': {'en': 'Account ID'},
              'type': 'text',
            },
            {
              'key': 'coins',
              'label': {'en': 'Enter Amount', 'ar': 'ادخل العدد المطلوب'},
              'type': 'amount',
              'constraints': {'min': 1000, 'max': 1000000},
            },
          ],
      'unitPricing': unitPricing ?? {'fieldKey': 'coins', 'blockUnits': 1000},
    };

void main() {
  group('unitLinePrice', () {
    test('scales the block price to the amount, rounded to cents', () {
      expect(unitLinePrice(0.45, 45000, 1000), 20.25);
      expect(unitLinePrice(1, 1500, 1000), 1.5);
      // 0.33 * 1001 / 1000 = 0.33033 → 0.33
      expect(unitLinePrice(0.33, 1001, 1000), 0.33);
      // half-cent rounds up: 0.5 * 25 / 1000 = 0.0125 → 0.01
      expect(unitLinePrice(0.5, 25, 1000), 0.01);
      expect(unitLinePrice(0.45, 1000, 0), 0);
    });

    test('roundCents', () {
      expect(roundCents(20.249999999), 20.25);
    });
  });

  group('ProductDto unitPricing', () {
    test('parses unitPricing and resolves the driving amount field', () {
      final p = ProductDto.fromJson(_coinsJson()).toEntity();

      expect(p.isUnitPriced, isTrue);
      expect(p.unitPricing!.fieldKey, 'coins');
      expect(p.unitPricing!.blockUnits, 1000.0);
      final field = p.unitAmountField;
      expect(field, isNotNull);
      expect(field!.key, 'coins');
      expect(field.label.resolve('ar'), 'ادخل العدد المطلوب');
      expect(field.constraints!.min, 1000.0);
      expect(p.unitLinePriceFor(45000), 20.25);
    });

    test('sub-cent amount lines price below \$0.01 (amountTooSmall gate)', () {
      final p = ProductDto.fromJson(_coinsJson()).toEntity();
      // 0.45 per 1000 → 10 units = 0.0045 → rounds to 0.00, which the product
      // page flags as amountTooSmall and the server rejects (< $0.01).
      expect(p.unitLinePriceFor(10)! < 0.01, isTrue);
      expect(p.unitLinePriceFor(23)! >= 0.01, isTrue);
    });

    test('from price = block price × min ÷ blockUnits', () {
      final p = ProductDto.fromJson(_coinsJson(
        unitPricing: {'fieldKey': 'coins', 'blockUnits': 1000},
        inputFields: [
          {
            'key': 'coins',
            'type': 'amount',
            'constraints': {'min': 5000, 'max': 100000},
          },
        ],
      )).toEntity();

      expect(p.fromPrice, 2.25); // 0.45 × 5000 / 1000
      expect(p.offerFromPrice, isNull);
    });

    test('offer/reseller price rescales the offer from price', () {
      final p = ProductDto.fromJson(_coinsJson(
        variants: [
          {'id': 'v1', 'denomination': '1000', 'price': 0.5, 'offerPrice': 0.4},
        ],
      )..['offer'] = {
              'discountType': 'percent',
              'discountValue': 20,
              'originalFromPrice': 0.5,
              'offerFromPrice': 0.4,
            }).toEntity();

      expect(p.fromPrice, 0.5); // min 1000 = one block
      expect(p.offerFromPrice, 0.4);
      // the line price uses the effective (offer) price
      expect(p.unitLinePriceFor(45000), 18.0);
    });

    test('absent unitPricing leaves a normal product', () {
      final json = _coinsJson()..remove('unitPricing');
      final p = ProductDto.fromJson(json).toEntity();
      expect(p.isUnitPriced, isFalse);
      expect(p.unitAmountField, isNull);
      expect(p.fromPrice, 0.45);
    });

    test('misconfigured spec (field missing / not amount / 2 variants) is unbuyable',
        () {
      final missing = ProductDto.fromJson(
              _coinsJson(unitPricing: {'fieldKey': 'nope', 'blockUnits': 1000}))
          .toEntity();
      expect(missing.isUnitPriced, isTrue);
      expect(missing.unitAmountField, isNull);
      expect(missing.unitLinePriceFor(1000), isNull);

      final notAmount = ProductDto.fromJson(_coinsJson(
              unitPricing: {'fieldKey': 'playerId', 'blockUnits': 1000}))
          .toEntity();
      expect(notAmount.unitAmountField, isNull);

      final twoVariants = ProductDto.fromJson(_coinsJson(variants: [
        {'id': 'v1', 'price': 0.45},
        {'id': 'v2', 'price': 0.9},
      ])).toEntity();
      expect(twoVariants.unitAmountField, isNull);
      // falls back to the normal lowest-variant from price
      expect(twoVariants.fromPrice, 0.45);
    });
  });

  group('amount lines', () {
    test('formatUnits groups thousands', () {
      expect(formatUnits(45000), '45,000');
      expect(formatUnits(1000000), '1,000,000');
      expect(formatUnits(999), '999');
    });

    test('cart key includes the amount', () {
      const base = CartItem(
        productId: 'p',
        variantId: 'v',
        title: I18nString({'en': 'x'}),
        category: 'c',
        variantLabel: '45,000',
        price: 20.25,
        qty: 1,
        fulfillmentType: 'account_credit',
        amount: 45000,
      );
      expect(base.key, 'p|v|45000.0|');
      expect(base.copyWith(playerId: 'id').key, 'p|v|45000.0|id');
      expect(base.isAmountLine, isTrue);
      expect(base.copyWith(playerId: 'id').amount, 45000);
      const plain = CartItem(
        productId: 'p',
        variantId: 'v',
        title: I18nString({'en': 'x'}),
        category: 'c',
        variantLabel: 'USD 5',
        price: 5,
        qty: 2,
        fulfillmentType: 'code',
      );
      expect(plain.key, 'p|v||');
      expect(plain.isAmountLine, isFalse);
    });

    test('place-order request sends amount only when set', () {
      final json = placeOrderRequestJson(const PlaceOrderInput(
        paymentMethod: 'wallet',
        items: [
          PlaceOrderLine(
            productId: 'p-uplive',
            variantId: 'v1',
            qty: 1,
            amount: 45000,
            playerId: '123456',
            fields: [PlaceOrderField(key: 'playerId', value: '123456')],
          ),
          PlaceOrderLine(productId: 'p-card', variantId: 'v9', qty: 2),
        ],
      ));
      final items = json['items'] as List;
      final amountLine = items[0] as Map<String, dynamic>;
      expect(amountLine['amount'], 45000.0);
      expect(amountLine['qty'], 1);
      expect(amountLine['variantId'], 'v1');
      expect(amountLine['fields'], [
        {'key': 'playerId', 'value': '123456'},
      ]);
      expect((items[1] as Map<String, dynamic>).containsKey('amount'), isFalse);
    });

    test('order item parses the optional amount', () {
      final withAmount = OrderItemDto.fromJson(<String, dynamic>{
        'productId': 'p',
        'variantId': 'v',
        'denomination': '45,000',
        'qty': 1,
        'price': 20.25,
        'amount': 45000,
      }).toEntity();
      expect(withAmount.amount, 45000.0);
      expect(withAmount.denomination, '45,000');

      final without = OrderItemDto.fromJson(<String, dynamic>{
        'productId': 'p',
        'variantId': 'v',
        'qty': 1,
        'price': 5,
      }).toEntity();
      expect(without.amount, isNull);
    });
  });
}
