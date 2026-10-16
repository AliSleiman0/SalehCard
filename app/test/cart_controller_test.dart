import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:salehcard_app/core/i18n/i18n_string.dart';
import 'package:salehcard_app/features/cart/domain/entities/cart_item.dart';
import 'package:salehcard_app/features/cart/presentation/controllers/cart_controller.dart';

CartItem _item({double? amount, String? playerId, int qty = 1}) => CartItem(
      productId: 'p',
      variantId: 'v',
      title: const I18nString({'en': 'x'}),
      category: 'c',
      variantLabel: amount != null ? '45,000' : 'USD 5',
      price: amount != null ? 20.25 : 5,
      qty: qty,
      fulfillmentType: 'account_credit',
      playerId: playerId,
      amount: amount,
    );

void main() {
  late ProviderContainer container;
  late CartController cart;

  setUp(() {
    container = ProviderContainer();
    cart = container.read(cartControllerProvider.notifier);
  });
  tearDown(() => container.dispose());

  List<CartItem> lines() => container.read(cartControllerProvider);

  test('same amount for different player IDs keeps separate lines', () {
    cart.add(_item(amount: 45000, playerId: 'A'));
    cart.add(_item(amount: 45000, playerId: 'B'));
    expect(lines(), hasLength(2));
    expect(lines().map((e) => e.playerId), ['A', 'B']);
    expect(lines().every((e) => e.qty == 1), isTrue);
  });

  test('same amount for the same player ID replaces, never doubles', () {
    cart.add(_item(amount: 45000, playerId: 'A'));
    cart.add(_item(amount: 45000, playerId: 'A'));
    expect(lines(), hasLength(1));
    expect(lines().single.qty, 1);
  });

  test('quantity lines for different player IDs stay separate', () {
    cart.add(_item(playerId: 'A', qty: 2));
    cart.add(_item(playerId: 'B', qty: 3));
    expect(lines(), hasLength(2));
    expect(lines().map((e) => '${e.playerId}:${e.qty}'), ['A:2', 'B:3']);
  });

  test('quantity lines for the same player ID merge quantity', () {
    cart.add(_item(playerId: 'A', qty: 2));
    cart.add(_item(playerId: 'A', qty: 3));
    expect(lines().single.qty, 5);
  });

  test('setQty and remove address one player line only', () {
    cart.add(_item(playerId: 'A'));
    cart.add(_item(playerId: 'B'));
    cart.setQty(lines().first.key, 4);
    expect(lines().map((e) => e.qty), [4, 1]);
    cart.remove(lines().last.key);
    expect(lines().single.playerId, 'A');
  });
}
