import { useState, useEffect } from "react";
import {
  View,
  Text,
  KeyboardAvoidingView,
  Platform,
  ScrollView,
} from "react-native";
import { Link, useRouter } from "expo-router";
import { Input } from "../../components/ui/Input";
import { Button } from "../../components/ui/Button";
import { useAuth } from "../../hooks/useAuth";
import { getApiErrorMessage } from "../../lib/api";
import { useGoogleAuth, extractIdToken } from "../../lib/googleAuth";
import axios from "axios";

export default function LoginScreen() {
  const { login, googleLogin } = useAuth();
  const router = useRouter();
  const { request, response, promptAsync } = useGoogleAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [googleLoading, setGoogleLoading] = useState(false);
  const [error, setError] = useState("");

  // Handle Google auth response
  useEffect(() => {
    const idToken = extractIdToken(response);
    if (!idToken) return;

    setGoogleLoading(true);
    setError("");

    googleLogin({ id_token: idToken })
      .catch((err) => {
        if (
          axios.isAxiosError(err) &&
          err.response?.data?.code === "MISSING_FIELDS"
        ) {
          // New user — redirect to register with the Google token
          router.push(
            `/(auth)/register?google_token=${encodeURIComponent(idToken)}`
          );
          return;
        }
        setError(getApiErrorMessage(err));
      })
      .finally(() => setGoogleLoading(false));
  }, [response]);

  const handleLogin = async () => {
    if (!email.trim() || !password.trim()) {
      setError("Completá todos los campos.");
      return;
    }
    setError("");
    setLoading(true);
    try {
      await login({ email: email.trim(), password });
    } catch (err) {
      setError(getApiErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  return (
    <KeyboardAvoidingView
      className="flex-1"
      behavior={Platform.OS === "ios" ? "padding" : "height"}
    >
      <ScrollView
        contentContainerClassName="flex-1 justify-center px-6"
        keyboardShouldPersistTaps="handled"
      >
        <View className="mb-8">
          <Text className="text-3xl font-bold text-gray-900">
            Iniciar sesión
          </Text>
          <Text className="text-base text-gray-500 mt-2">
            Ingresá a tu cuenta de Turnero
          </Text>
        </View>

        {error ? (
          <View className="bg-red-50 border border-red-200 rounded-xl p-3 mb-4">
            <Text className="text-red-600 text-sm">{error}</Text>
          </View>
        ) : null}

        <Button
          title="Continuar con Google"
          variant="outline"
          onPress={() => promptAsync()}
          loading={googleLoading}
          disabled={!request}
          className="mb-4"
        />

        <View className="flex-row items-center mb-4">
          <View className="flex-1 h-px bg-gray-200" />
          <Text className="mx-4 text-gray-400 text-sm">o</Text>
          <View className="flex-1 h-px bg-gray-200" />
        </View>

        <Input
          label="Email"
          placeholder="tu@email.com"
          value={email}
          onChangeText={setEmail}
          keyboardType="email-address"
          autoCapitalize="none"
          autoCorrect={false}
        />

        <Input
          label="Contraseña"
          placeholder="Tu contraseña"
          value={password}
          onChangeText={setPassword}
          secureTextEntry
        />

        <Button
          title="Ingresar"
          onPress={handleLogin}
          loading={loading}
          className="mt-2"
        />

        <View className="flex-row justify-center mt-6">
          <Text className="text-gray-500">¿No tenés cuenta? </Text>
          <Link href="/(auth)/register" asChild>
            <Text className="text-primary-600 font-semibold">
              Registrate
            </Text>
          </Link>
        </View>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}
