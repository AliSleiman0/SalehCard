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

/// Payload for a top-up REQUEST: the wallet is credited only after an admin
/// confirms the out-of-band payment ([channel]) and approves the request.
class TopUpInput {
  const TopUpInput({required this.amount, required this.channel, this.note});

  final double amount;
  final String channel; // usdt | whish | omt | cash | other
  final String? note;
}

/// Moderation state of a top-up request.
enum TopUpStatus { pending, approved, rejected, unknown }

TopUpStatus topUpStatusFromString(String value) {
  switch (value) {
    case 'pending':
      return TopUpStatus.pending;
    case 'approved':
      return TopUpStatus.approved;
    case 'rejected':
      return TopUpStatus.rejected;
    default:
      return TopUpStatus.unknown;
  }
}

/// A customer's top-up request and its review outcome.
class TopUpRequest {
  const TopUpRequest({
    required this.id,
    required this.amount,
    required this.channel,
    required this.status,
    this.note = '',
    this.decisionReason = '',
    this.createdAt,
  });

  final String id;
  final double amount;
  final String channel;
  final TopUpStatus status;
  final String note;
  final String decisionReason;
  final DateTime? createdAt;
}
