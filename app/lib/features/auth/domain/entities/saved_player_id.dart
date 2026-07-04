/// A player / account ID the customer has saved for faster checkout, tagged
/// with a human-readable [label] (e.g. "PUBG main"). Both fields are required.
class SavedPlayerId {
  const SavedPlayerId({required this.label, required this.value});

  /// Friendly name the customer gave this ID.
  final String label;

  /// The actual player/account ID passed to the store at checkout.
  final String value;

  @override
  bool operator ==(Object other) =>
      other is SavedPlayerId && other.label == label && other.value == value;

  @override
  int get hashCode => Object.hash(label, value);
}
