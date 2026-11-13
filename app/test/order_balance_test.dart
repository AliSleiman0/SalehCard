import 'package:flutter_test/flutter_test.dart';
import 'package:salehcard_app/core/i18n/arb/app_localizations_en.dart';
import 'package:salehcard_app/features/checkout/data/dtos/order_dto.dart';
import 'package:salehcard_app/features/orders/presentation/widgets/balance_line.dart';

void main() {
  final l10n = AppLocalizationsEn();

  test('wallet balances are parsed and rendered before → after', () {
    final o = OrderDto.fromJson(<String, dynamic>{
      'id': 'o1',
      'paymentMethod': 'wallet',
      'walletBalanceBefore': 100,
      'walletBalanceAfter': 87.7,
      'refundBalanceAfter': 100,
    }).toEntity();

    expect(o.walletBalanceBefore, 100.0);
    expect(o.walletBalanceAfter, 87.7);
    expect(balanceLineText(o, l10n),
        'Balance: \$100.00 → \$87.70 · Refunded → \$100.00');
  });

  test('no balances → no line', () {
    final o = OrderDto.fromJson(<String, dynamic>{'id': 'o1', 'paymentMethod': 'usdt'})
        .toEntity();
    expect(balanceLineText(o, l10n), isNull);
  });
}
