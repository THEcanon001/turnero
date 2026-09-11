import { View, Text, SafeAreaView, TouchableOpacity } from "react-native";
import { useRouter } from "expo-router";
import { Card } from "../../../components/ui/Card";

const MENU_ITEMS = [
  { title: "Facturación", route: "/(provider)/more/billing", icon: "💰" },
  { title: "Código QR", route: "/(provider)/more/qr", icon: "📱" },
  { title: "Horarios", route: "/(provider)/more/schedule", icon: "🕐" },
  { title: "Configuración", route: "/(provider)/more/settings", icon: "⚙️" },
];

export default function MoreScreen() {
  const router = useRouter();

  return (
    <SafeAreaView className="flex-1 bg-gray-50">
      <View className="px-4 pt-4">
        {MENU_ITEMS.map((item) => (
          <TouchableOpacity
            key={item.route}
            onPress={() => router.push(item.route as never)}
          >
            <Card className="mb-3">
              <View className="flex-row items-center">
                <Text className="text-2xl mr-4">{item.icon}</Text>
                <Text className="text-base font-semibold text-gray-800">
                  {item.title}
                </Text>
              </View>
            </Card>
          </TouchableOpacity>
        ))}
      </View>
    </SafeAreaView>
  );
}
