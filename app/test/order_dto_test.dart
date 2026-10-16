import 'package:flutter_test/flutter_test.dart';
import 'package:salehcard_app/features/checkout/data/dtos/order_dto.dart';
import 'package:salehcard_app/features/checkout/domain/entities/order.dart';

void main() {
  group('OrderDto.toEntity', () {
    test('maps a completed code order with a delivered code', () {
      final dto = OrderDto.fromJson(<String, dynamic>{
        'id': '665f000000000000000000aa',
        'userId': 'u1',
        'items': [
          {
            'productId': 'p1',
            'variantId': 'v1',
            'title': {'en': 'Roblox', 'ar': 'روبلوكس', 'tr': 'Roblox'},
            'denomination': 'USD 10',
            'category': 'games',
            'qty': 2,
            'price': 10.0,
            'fulfillmentType': 'code',
          },
        ],
        'subtotal': 20.0,
        'total': 20.0,
        'currency': 'USD',
        'paymentMethod': 'wallet',
        'status': 'completed',
        'fulfillment': {
          'deliveredCode': 'RBLX-7K2M',
          'statusTimeline': [
            {'status': 'created', 'note': '', 'at': '2026-06-25T10:00:00Z'},
            {'status': 'delivered', 'note': '', 'at': '2026-06-25T10:00:01Z'},
          ],
        },
        'createdAt': '2026-06-25T10:00:00Z',
        'updatedAt': '2026-06-25T10:00:01Z',
      });

      final order = dto.toEntity();

      expect(order.status, OrderStatus.completed);
      expect(order.isCompleted, isTrue);
      expect(order.hasDeliveredCode, isTrue);
      expect(order.fulfillment.deliveredCode, 'RBLX-7K2M');
      expect(order.fulfillment.statusTimeline.length, 2);
      expect(order.items.single.title.resolve('ar'), 'روبلوكس');
      expect(order.items.single.qty, 2);
      expect(order.total, 20.0);
      expect(order.reference, '0000AA');
    });

    test('maps a processing order with no delivered code', () {
      final dto = OrderDto.fromJson(<String, dynamic>{
        'id': 'p2',
        'status': 'processing',
        'paymentMethod': 'card',
        'fulfillment': {'creditedToId': 'player-123'},
      });

      final order = dto.toEntity();

      expect(order.status, OrderStatus.processing);
      expect(order.isProcessing, isTrue);
      expect(order.hasDeliveredCode, isFalse);
      expect(order.fulfillment.creditedToId, 'player-123');
      expect(order.items, isEmpty);
    });
  });

  group('per-line codes', () {
    Map<String, dynamic> item(String title, String type,
            {List<String>? codes, int qty = 1, String denom = ''}) =>
        <String, dynamic>{
          'productId': 'p-$title',
          'title': {'en': title},
          'denomination': denom,
          'qty': qty,
          'price': 5.0,
          'fulfillmentType': type,
          'codes': ?codes,
        };

    Order parse(String status, List<Map<String, dynamic>> items,
            {String? deliveredCode}) =>
        OrderDto.fromJson(<String, dynamic>{
          'id': 'o1',
          'status': status,
          'items': items,
          'fulfillment': {'deliveredCode': ?deliveredCode},
        }).toEntity();

    test('item DTO parses codes when present and defaults to empty', () {
      final withCodes = OrderItemDto.fromJson(
              item('Google Play', 'code', codes: ['A-1', 'A-2'], qty: 2))
          .toEntity();
      expect(withCodes.codes, ['A-1', 'A-2']);

      final without = OrderItemDto.fromJson(item('LAKA', 'account_credit'))
          .toEntity();
      expect(without.codes, isEmpty);
    });

    test('mixed processing order exposes the code line + manual lines', () {
      final order = parse(
        'processing',
        [
          item('LAKA', 'account_credit'),
          item('Google Play', 'code', codes: ['GP-1'], denom: 'USD 25'),
        ],
        deliveredCode: 'GP-1',
      );
      expect(order.isProcessing, isTrue);
      expect(order.hasManualLines, isTrue);
      final lines = order.codeLines;
      expect(lines, hasLength(1));
      expect(lines.single.codes, ['GP-1']);
      expect(lines.single.label('en'), 'Google Play · USD 25');
    });

    test('pure code order with several lines gives one entry per line', () {
      final order = parse(
        'completed',
        [
          item('Roblox', 'code', codes: ['R-1', 'R-2'], qty: 2),
          item('Steam', 'code', codes: ['S-1']),
        ],
        deliveredCode: 'R-1',
      );
      expect(order.hasManualLines, isFalse);
      expect(order.codeLines.map((l) => l.codes).toList(), [
        ['R-1', 'R-2'],
        ['S-1'],
      ]);
      expect(order.codeLines.map((l) => l.label('en')).toList(),
          ['Roblox', 'Steam']);
    });

    test('old response shape falls back to deliveredCode + first code item',
        () {
      final order = parse(
        'completed',
        [
          item('LAKA', 'account_credit'),
          item('Roblox', 'code', qty: 2),
        ],
        deliveredCode: 'RBLX-OLD',
      );
      final lines = order.codeLines;
      expect(lines, hasLength(1));
      expect(lines.single.codes, ['RBLX-OLD']);
      expect(lines.single.label('en'), 'Roblox');
    });

    test('pure credit order has no code lines', () {
      final order = parse('processing', [item('LAKA', 'account_credit')]);
      expect(order.codeLines, isEmpty);
      expect(order.hasManualLines, isTrue);
    });
  });
}
