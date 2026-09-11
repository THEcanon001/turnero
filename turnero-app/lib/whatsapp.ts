import { Linking } from "react-native";

export function openWhatsApp(phone: string, message?: string): void {
  // Strip non-numeric characters
  const cleanPhone = phone.replace(/\D/g, "");
  let url = `https://wa.me/${cleanPhone}`;
  if (message) {
    url += `?text=${encodeURIComponent(message)}`;
  }
  Linking.openURL(url);
}
