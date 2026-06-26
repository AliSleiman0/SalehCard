/// Kind of a wallet ledger entry (mirrors the backend transaction `type`).
enum WalletTxType { topup, purchase, refund, adjustment, unknown }

WalletTxType walletTxTypeFromString(String value) {
  switch (value) {
    case 'topup':
      return WalletTxType.topup;
    case 'purchase':
      return WalletTxType.purchase;
    case 'refund':
      return WalletTxType.refund;
    case 'adjustment':
      return WalletTxType.adjustment;
    default:
      return WalletTxType.unknown;
  }
}

/// A single wallet ledger entry. [amount] is SIGNED: positive = credit
/// (topup/refund), negative = debit (purchase). [balanceAfter] is the running
/// balance immediately after this transaction.
class WalletTx {
  const WalletTx({
    required this.id,
    required this.type,
    required this.amount,
    required this.balanceAfter,
    required this.method,
    required this.ref,
    this.createdAt,
  });

  final String id;
  final WalletTxType type;
  final double amount;
  final double balanceAfter;
  final String method;
  final String ref;
  final DateTime? createdAt;

  bool get isCredit => amount >= 0;
}

/// The customer's wallet: current balance + newest-first transaction history.
class Wallet {
  const Wallet({required this.balance, required this.transactions});

  final double balance;
  final List<WalletTx> transactions;
}

/// Payload for a top-up. The server credits the balance immediately (card and
/// usdt are both mock-approved in dev); [ref] is an optional external reference.
class TopUpInput {
  const TopUpInput({required this.amount, required this.method, this.ref});

  final double amount;
  final String method; // card | usdt
  final String? ref;
}
