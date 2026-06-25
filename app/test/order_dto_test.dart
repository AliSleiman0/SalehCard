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
}
