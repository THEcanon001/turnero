import { useState, useEffect } from "react";
import {
  View,
  Text,
  KeyboardAvoidingView,
  Platform,
  ScrollView,
} from "react-native";
import { Link, useLocalSearchParams, useRouter } from "expo-router";
import { Input } from "../../components/ui/Input";
import { Button } from "../../components/ui/Button";
import { useAuth } from "../../hooks/useAuth";
import { getApiErrorMessage } from "../../lib/api";
import { useGoogleAuth, extractIdToken } from "../../lib/googleAuth";
import axios from "axios";

type AccountType = "individual" | "business";

export default function RegisterScreen() {
  const { register, googleLogin } = useAuth();
  const router = useRouter();
  const params = useLocalSearchParams<{ google_token?: string }>();
  const { request, response, promptAsync } = useGoogleAuth();

  // If we arrive here from login with a google_token, show only the
  // missing fields form (slug, phone, type)
  const [googleToken, setGoogleToken] = useState(params.google_token ?? "");
  const isGoogleFlow = !!googleToken;

  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [phone, setPhone] = useState("");
  const [slug, setSlug] = useState("");
  const [accountType, setAccountType] = useState<AccountType>("individual");
  const [businessName, setBusinessName] = useState("");
  const [loading, setLoading] = useState(false);
  const [googleLoading, setGoogleLoading] = useState(false);
  const [error, setError] = useState("");

  // Handle Google auth response (when tapping Google button on register screen)
  useEffect(() => {
    const idToken = extractIdToken(response);
    if (!idToken) return;

    setGoogleLoading(true);
    setError("");

    googleLogin({ id_token: idToken })
      .then(() => {
        // Returning user — already logged in
      })
      .catch((err) => {
        if (
          axios.isAxiosError(err) &&
          err.response?.data?.code === "MISSING_FIELDS"
        ) {
          // New user — show the missing fields form
          setGoogleToken(idToken);
          return;
        }
        setError(getApiErrorMessage(err));
      })
      .finally(() => setGoogleLoading(false));
  }, [response]);

  const handleRegister = async () => {
    if (!isGoogleFlow) {
      // Standard email/password registration
      if (
        !name.trim() ||
        !email.trim() ||
        !password ||
        !phone.trim() ||
        !slug.trim()
      ) {
        setError("Completá todos los campos obligatorios.");
        return;
      }
      if (password.length < 8) {
        setError("La contraseña debe tener al menos 8 caracteres.");
        return;
      }
    } else {
      // Google flow — only need slug + phone
      if (!phone.trim() || !slug.trim()) {
        setError("Completá tu teléfono y slug.");
        return;
      }
    }

    if (accountType === "business" && !businessName.trim()) {
      setError("Ingresá el nombre de tu negocio.");
      return;
    }

    setError("");
    setLoading(true);
    try {
      if (isGoogleFlow) {
        await googleLogin({
          id_token: googleToken,
          type: accountType,
          slug: slug.trim().toLowerCase(),
          phone: phone.trim(),
          timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
          business_name:
            accountType === "business" ? businessName.trim() : undefined,
        });
      } else {
        await register({
          name: name.trim(),
          email: email.trim(),
          password,
          phone: phone.trim(),
          type: accountType,
          slug: slug.trim().toLowerCase(),
          business_name:
            accountType === "business" ? businessName.trim() : null,
          timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
        });
      }
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
        contentContainerClassName="py-12 px-6"
        keyboardShouldPersistTaps="handled"
      >
        <View className="mb-8">
          <Text className="text-3xl font-bold text-gray-900">
            {isGoogleFlow ? "Completá tu perfil" : "Crear cuenta"}
          </Text>
          <Text className="text-base text-gray-500 mt-2">
            {isGoogleFlow
              ? "Solo necesitamos algunos datos más"
              : "Registrate para gestionar tus turnos"}
          </Text>
        </View>

        {error ? (
          <View className="bg-red-50 border border-red-200 rounded-xl p-3 mb-4">
            <Text className="text-red-600 text-sm">{error}</Text>
          </View>
        ) : null}

        {/* Google button only shown if NOT in Google completion flow */}
        {!isGoogleFlow && (
          <>
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
          </>
        )}

        {/* Standard fields — hidden in Google flow */}
        {!isGoogleFlow && (
          <>
            <Input
              label="Nombre"
              placeholder="Tu nombre completo"
              value={name}
              onChangeText={setName}
            />

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
              placeholder="Mínimo 8 caracteres"
              value={password}
              onChangeText={setPassword}
              secureTextEntry
            />
          </>
        )}

        {/* Always shown fields */}
        <Input
          label="Teléfono"
          placeholder="+54 11 1234 5678"
          value={phone}
          onChangeText={setPhone}
          keyboardType="phone-pad"
        />

        <Input
          label="Slug (URL de tu perfil)"
          placeholder="mi-negocio"
          value={slug}
          onChangeText={(text) =>
            setSlug(text.toLowerCase().replace(/[^a-z0-9-]/g, ""))
          }
          autoCapitalize="none"
          autoCorrect={false}
        />

        {/* Account type selector */}
        <Text className="text-sm font-medium text-gray-700 mb-2">
          Tipo de cuenta
        </Text>
        <View className="flex-row gap-3 mb-4">
          <Button
            title="Personal"
            variant={accountType === "individual" ? "primary" : "outline"}
            onPress={() => setAccountType("individual")}
            className="flex-1"
          />
          <Button
            title="Negocio"
            variant={accountType === "business" ? "primary" : "outline"}
            onPress={() => setAccountType("business")}
            className="flex-1"
          />
        </View>

        {accountType === "business" && (
          <Input
            label="Nombre del negocio"
            placeholder="Mi Peluquería"
            value={businessName}
            onChangeText={setBusinessName}
          />
        )}

        <Button
          title={isGoogleFlow ? "Completar registro" : "Registrarme"}
          onPress={handleRegister}
          loading={loading}
          className="mt-2"
        />

        <View className="flex-row justify-center mt-6">
          <Text className="text-gray-500">¿Ya tenés cuenta? </Text>
          <Link href="/(auth)/login" asChild>
            <Text className="text-primary-600 font-semibold">
              Iniciá sesión
            </Text>
          </Link>
        </View>
      </ScrollView>
    </KeyboardAvoidingView>
  );
}
