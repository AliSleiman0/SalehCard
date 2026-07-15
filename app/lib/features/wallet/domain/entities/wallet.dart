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
/// confirms the out-of-band payment and approves the request. A request carries
/// either an admin-defined [methodId] (with its collected [fields]) or, for
/// legacy flows, a bare [channel].
class TopUpInput {
  const TopUpInput({
    required this.amount,
    this.channel = '',
    this.methodId,
    this.fields = const {},
    this.note,
  });

  final double amount;
  final String channel;
  final String? methodId;
  final Map<String, String> fields; // key -> value (file fields hold a URL)
  final String? note;
}

/// The input kind a manual top-up method collects from the customer.
enum TopUpFieldType { text, number, select, file, unknown }

TopUpFieldType topUpFieldTypeFromString(String value) {
  switch (value) {
    case 'text':
      return TopUpFieldType.text;
    case 'number':
      return TopUpFieldType.number;
    case 'select':
      return TopUpFieldType.select;
    case 'file':
      return TopUpFieldType.file;
    default:
      return TopUpFieldType.unknown;
  }
}

/// One customer-input spec on a manual top-up method.
class TopUpMethodField {
  const TopUpMethodField({
    required this.key,
    required this.label,
    required this.type,
    this.isRequired = false,
    this.options = const [],
    this.placeholder = '',
  });

  final String key;
  final String label;
  final TopUpFieldType type;
  final bool isRequired;
  final List<String> options;
  final String placeholder;
}

/// An admin-defined manual funding method shown on the top-up screen.
class TopUpMethod {
  const TopUpMethod({
    required this.id,
    required this.name,
    this.instructions = '',
    this.fields = const [],
  });

  final String id;
  final String name;
  final String instructions;
  final List<TopUpMethodField> fields;
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
    this.methodName = '',
    this.note = '',
    this.decisionReason = '',
    this.createdAt,
  });

  final String id;
  final double amount;
  final String channel;
  final String methodName;
  final TopUpStatus status;
  final String note;
  final String decisionReason;
  final DateTime? createdAt;

  /// Human label for the request's funding method (admin method name, else the
  /// legacy channel).
  String get methodLabel => methodName.isNotEmpty ? methodName : channel;
}
