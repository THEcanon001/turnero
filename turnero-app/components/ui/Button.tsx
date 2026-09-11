import {
  TouchableOpacity,
  Text,
  ActivityIndicator,
  type TouchableOpacityProps,
} from "react-native";

interface ButtonProps extends TouchableOpacityProps {
  title: string;
  variant?: "primary" | "secondary" | "outline" | "danger";
  loading?: boolean;
}

const variantStyles = {
  primary: "bg-primary-600 active:bg-primary-700",
  secondary: "bg-gray-200 active:bg-gray-300",
  outline: "border border-primary-600 bg-transparent",
  danger: "bg-red-600 active:bg-red-700",
};

const textStyles = {
  primary: "text-white",
  secondary: "text-gray-800",
  outline: "text-primary-600",
  danger: "text-white",
};

export function Button({
  title,
  variant = "primary",
  loading = false,
  disabled,
  className,
  ...props
}: ButtonProps) {
  return (
    <TouchableOpacity
      className={`py-3 px-6 rounded-xl items-center justify-center ${variantStyles[variant]} ${
        disabled || loading ? "opacity-50" : ""
      } ${className ?? ""}`}
      disabled={disabled || loading}
      {...props}
    >
      {loading ? (
        <ActivityIndicator
          color={variant === "secondary" ? "#374151" : "#FFFFFF"}
        />
      ) : (
        <Text
          className={`text-base font-semibold ${textStyles[variant]}`}
        >
          {title}
        </Text>
      )}
    </TouchableOpacity>
  );
}
