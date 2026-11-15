/// The order number customers and staff see everywhere (admin, web, app):
/// "#" + the last 8 characters of the order id, upper-cased — e.g. "#8D03458D".
/// Admin order search accepts it as typed (with or without the #).
String orderNumber(String id) {
  final tail = id.length <= 8 ? id : id.substring(id.length - 8);
  return '#${tail.toUpperCase()}';
}
