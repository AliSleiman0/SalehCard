import 'package:flutter/material.dart';

import '../../../core/i18n/arb/app_localizations.dart';
import '../../../core/theme/app_tokens.dart';
import '../../../core/widgets/status_badge.dart';
import '../domain/complaint.dart';

/// Localized label + badge colour for a complaint status.
(String, Color) complaintStatusLabel(ComplaintStatus s, AppLocalizations l10n) {
  switch (s) {
    case ComplaintStatus.open:
      return (l10n.complaintStatusOpen, StatusBadge.warning);
    case ComplaintStatus.inProgress:
      return (l10n.complaintStatusInProgress, AppTokens.brand1);
    case ComplaintStatus.resolved:
      return (l10n.complaintStatusResolved, StatusBadge.success);
    case ComplaintStatus.closed:
    case ComplaintStatus.unknown:
      return (l10n.complaintStatusClosed, StatusBadge.neutral);
  }
}

/// Localized label for a complaint category key.
String complaintCategoryLabel(String key, AppLocalizations l10n) {
  switch (key) {
    case 'order':
      return l10n.complaintCatOrder;
    case 'topup':
      return l10n.complaintCatTopup;
    case 'account':
      return l10n.complaintCatAccount;
    default:
      return l10n.complaintCatOther;
  }
}
