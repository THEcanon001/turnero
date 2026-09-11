import { View, Text } from "react-native";

interface EmptyStateProps {
  title: string;
  message?: string;
}

export function EmptyState({ title, message }: EmptyStateProps) {
  return (
    <View className="flex-1 items-center justify-center p-8">
      <Text className="text-lg font-semibold text-gray-500 text-center">
        {title}
      </Text>
      {message && (
        <Text className="text-sm text-gray-400 text-center mt-2">
          {message}
        </Text>
      )}
    </View>
  );
}
