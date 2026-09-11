import { Tabs } from "expo-router";
import { Text } from "react-native";

export default function EmployeeLayout() {
  return (
    <Tabs
      screenOptions={{
        headerStyle: { backgroundColor: "#FFFFFF" },
        headerTintColor: "#111827",
        headerTitleStyle: { fontWeight: "600" },
        tabBarActiveTintColor: "#4F46E5",
        tabBarInactiveTintColor: "#9CA3AF",
        tabBarStyle: { backgroundColor: "#FFFFFF" },
      }}
    >
      <Tabs.Screen
        name="agenda"
        options={{
          title: "Mi Agenda",
          headerShown: false,
          tabBarIcon: ({ color }) => (
            <Text style={{ color, fontSize: 20 }}>📅</Text>
          ),
        }}
      />
      <Tabs.Screen
        name="schedule"
        options={{
          title: "Mi Horario",
          tabBarIcon: ({ color }) => (
            <Text style={{ color, fontSize: 20 }}>🕐</Text>
          ),
        }}
      />
    </Tabs>
  );
}
