/// The four KYC verification states a customer can be in. Drives which status
/// card the `/kyc` screen renders and the [StatusBadge] colour.
enum KycStatus { unverified, pending, verified, rejected }

/// The document types a customer can submit for verification.
enum KycDocumentType { passport, idCard, license }

/// The form payload submitted for verification. A plain value object — the
/// [documentFileName] is a placeholder for the faux upload (there is no real
/// file picker / upload yet, see the form screen + stub).
class KycSubmission {
  const KycSubmission({
    required this.fullName,
    required this.documentType,
    required this.documentNumber,
    required this.documentFileName,
  });

  final String fullName;
  final KycDocumentType documentType;
  final String documentNumber;

  /// Placeholder for the uploaded document (e.g. "document.jpg"). There is no
  /// real upload yet — this just records that the user picked a file.
  final String documentFileName;
}

/// Optional richer profile. Today only [status] is meaningful; [rejectionReason]
/// lets the rejected card show a reason when one exists.
class KycProfile {
  const KycProfile({required this.status, this.rejectionReason});

  final KycStatus status;
  final String? rejectionReason;
}
